package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

type fakeLeaderboardRepo struct {
	calls   int
	ranking func(start, end time.Time) []LeaderboardModelRow
	buckets func(bucket string) []LeaderboardBucketPoint
	months  []LeaderboardMonthTotal
}

func (f *fakeLeaderboardRepo) GetModelRanking(_ context.Context, start, end time.Time, _ string) ([]LeaderboardModelRow, error) {
	f.calls++
	return f.ranking(start, end), nil
}

func (f *fakeLeaderboardRepo) GetTotals(_ context.Context, start, end time.Time, _ string) (*LeaderboardTotals, error) {
	t := &LeaderboardTotals{}
	for _, r := range f.ranking(start, end) {
		t.Requests += r.Requests
		t.TotalTokens += r.TotalTokens
		t.Cost += r.Cost
		t.ActualCost += r.ActualCost
		t.Models++
	}
	return t, nil
}

func (f *fakeLeaderboardRepo) GetBucketedModelUsage(_ context.Context, _, _ time.Time, _, bucket string) ([]LeaderboardBucketPoint, error) {
	return f.buckets(bucket), nil
}

func (f *fakeLeaderboardRepo) GetMonthlyTotals(_ context.Context, _ string) ([]LeaderboardMonthTotal, error) {
	return f.months, nil
}

func TestModelLeaderboard_RankingGrowthAndRedaction(t *testing.T) {
	loc := timezone.Location()
	now := time.Date(2026, 9, 15, 12, 0, 0, 0, loc)
	monthStart := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
	prevStart := time.Date(2026, 8, 1, 0, 0, 0, 0, loc)

	repo := &fakeLeaderboardRepo{
		ranking: func(start, end time.Time) []LeaderboardModelRow {
			switch {
			case start.Equal(monthStart): // 当月
				return []LeaderboardModelRow{
					{Model: "b", Requests: 50, TotalTokens: 500, Cost: 2, ActualCost: 1},
					{Model: "a", Requests: 150, TotalTokens: 900, Cost: 6, ActualCost: 3},
					{Model: "new", Requests: 50, TotalTokens: 600},
				}
			case start.Equal(prevStart): // 上月同期
				// 对比期必须止于上月同一时刻（8/15 12:00），而不是整月
				if want := time.Date(2026, 8, 15, 12, 0, 0, 0, loc); !end.Equal(want) {
					t.Fatalf("prev period end = %v, want %v", end, want)
				}
				return []LeaderboardModelRow{
					{Model: "b", Requests: 100},
					{Model: "a", Requests: 100, TotalTokens: 1},
				}
			case end.Equal(monthStart): // 截至上月末累计
				return []LeaderboardModelRow{{Model: "b", Requests: 1000}, {Model: "a", Requests: 10}}
			default: // 历史累计
				return []LeaderboardModelRow{{Model: "b", Requests: 1050}, {Model: "a", Requests: 160}, {Model: "new", Requests: 50}}
			}
		},
		buckets: func(bucket string) []LeaderboardBucketPoint {
			if bucket == "day" {
				return []LeaderboardBucketPoint{{Bucket: "2026-09-01", Model: "a", Requests: 7}, {Bucket: "2026-09-15", Model: "b", Requests: 3}}
			}
			return []LeaderboardBucketPoint{{Bucket: "2026-08", Model: "b", Requests: 1000}, {Bucket: "2026-09", Model: "a", Requests: 150}, {Bucket: "2026-09", Model: "b", Requests: 50}}
		},
		months: []LeaderboardMonthTotal{{Month: "2026-08", Requests: 1010}, {Month: "2026-09", Requests: 250, Cost: 8}},
	}
	svc := NewModelLeaderboardService(repo)
	svc.now = func() time.Time { return now }

	resp, err := svc.Get(context.Background(), "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	r := resp.Monthly.Ranking
	if len(r) != 3 || r[0].Model != "a" || r[1].Model != "new" || r[2].Model != "b" {
		t.Fatalf("monthly order wrong: %+v", r)
	}
	// 50 请求并列时按 tokens 降序：new(600) 在 b(500) 之前
	if r[0].PrevRank == nil || *r[0].PrevRank != 1 || r[0].RequestsGrowth == nil || *r[0].RequestsGrowth != 0.5 {
		t.Fatalf("a growth/prev rank wrong: %+v", r[0])
	}
	if r[1].PrevRank != nil || r[1].RequestsGrowth != nil {
		t.Fatalf("new model should have no prev rank: %+v", r[1])
	}
	if r[2].RequestsGrowth == nil || *r[2].RequestsGrowth != -0.5 {
		t.Fatalf("b growth wrong: %+v", r[2])
	}
	if got := r[0].RequestShare; got != 0.6 {
		t.Fatalf("share = %v, want 0.6", got)
	}
	if len(resp.Monthly.Trend.Labels) != 15 || resp.Monthly.Trend.Labels[14] != "2026-09-15" {
		t.Fatalf("daily labels wrong: %v", resp.Monthly.Trend.Labels)
	}
	if resp.Monthly.Trend.Datasets[0].Model != "a" || resp.Monthly.Trend.Datasets[0].Requests[0] != 7 {
		t.Fatalf("daily dataset wrong: %+v", resp.Monthly.Trend.Datasets)
	}
	// 历史累计：对比“截至上月末”，a 从第 2 名保持第 2
	at := resp.AllTime.Ranking
	if at[0].Model != "b" || at[1].PrevRank == nil || *at[1].PrevRank != 2 {
		t.Fatalf("all-time ranking wrong: %+v", at)
	}
	if len(resp.MonthRows) != 2 || resp.MonthRows[0].Month != "2026-09" || resp.MonthRows[0].TopModel != "a" || resp.MonthRows[0].TopModelShare != 0.6 {
		t.Fatalf("month rows wrong: %+v", resp.MonthRows)
	}
	if resp.Months[0] != "2026-09" || resp.Monthly.Summary.Cost == nil || *resp.Monthly.Summary.Cost != 8 {
		t.Fatalf("months/cost wrong: %+v %+v", resp.Months, resp.Monthly.Summary)
	}

	// 非管理员：命中缓存，且费用字段被抹掉；缓存原件不受影响
	calls := repo.calls
	user, err := svc.Get(context.Background(), "requested", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if repo.calls != calls {
		t.Fatalf("expected cache hit, repo called again")
	}
	if user.CostVisible || user.Monthly.Summary.Cost != nil || user.Monthly.Prev.Cost != nil || user.Monthly.Ranking[0].Cost != nil || user.AllTime.Ranking[0].ActualCost != nil || user.MonthRows[0].Cost != nil {
		t.Fatalf("cost not redacted for non-admin")
	}
	again, _ := svc.Get(context.Background(), "requested", "", true)
	if again.Monthly.Ranking[0].Cost == nil {
		t.Fatalf("redaction mutated cached response")
	}
}

func TestModelLeaderboard_BucketSeriesOther(t *testing.T) {
	points := make([]LeaderboardBucketPoint, 0)
	for i := 0; i < leaderboardTrendTopN+3; i++ {
		points = append(points, LeaderboardBucketPoint{Bucket: "2026-09", Model: string(rune('a' + i)), Requests: int64(100 - i), TotalTokens: 1})
	}
	s := lbBuildBucketSeries(points, []string{"2026-08", "2026-09"})
	if len(s.Datasets) != leaderboardTrendTopN+1 {
		t.Fatalf("datasets = %d", len(s.Datasets))
	}
	other := s.Datasets[len(s.Datasets)-1]
	if other.Model != leaderboardOther || other.Requests[1] != int64(100-8)+int64(100-9)+int64(100-10) || other.Tokens[1] != 3 || other.Requests[0] != 0 {
		t.Fatalf("other bucket wrong: %+v", other)
	}
}

func TestParseLeaderboardMonth(t *testing.T) {
	now := time.Date(2026, 12, 31, 23, 0, 0, 0, timezone.Location())
	s, e, err := ParseLeaderboardMonth("", now)
	if err != nil || s.Month() != 12 || e.Year() != 2027 || e.Month() != 1 {
		t.Fatalf("current month: %v %v %v", s, e, err)
	}
	if _, _, err := ParseLeaderboardMonth("2026-13", now); err == nil {
		t.Fatal("expected invalid month error")
	}
	if _, _, err := ParseLeaderboardMonth("2026-09'; drop", now); err == nil {
		t.Fatal("expected invalid month error")
	}
}
