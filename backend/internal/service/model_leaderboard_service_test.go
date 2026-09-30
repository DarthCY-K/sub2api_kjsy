package service

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// fakeLeaderboardRepo 按区间返回预置数据，并按「方法:区间类别」记录调用次数。
type fakeLeaderboardRepo struct {
	mu        sync.Mutex
	calls     map[string]int
	boundary  time.Time // 当月月初：end == boundary 且 start 很早 = 历史快照查询
	stats     map[string][]LeaderboardModelStat
	users     map[string]map[string]int64
	pairs     map[string][]LeaderboardModelUser
	daily     map[string][]LeaderboardBucketPoint
	histGate  chan struct{} // 非 nil 时历史 stats 查询阻塞到 close
	prevCheck func(start, end time.Time)
}

func (f *fakeLeaderboardRepo) kind(start, end time.Time) string {
	switch {
	case start.Year() < 2001:
		return "hist"
	case start.Equal(f.boundary.AddDate(0, -1, 0)) && end.Before(f.boundary):
		return "prev"
	default:
		return start.Format("2006-01")
	}
}

func (f *fakeLeaderboardRepo) hit(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[name]++
}

func (f *fakeLeaderboardRepo) count(name string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[name]
}

func (f *fakeLeaderboardRepo) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, v := range f.calls {
		n += v
	}
	return n
}

func (f *fakeLeaderboardRepo) GetMonthModelStats(_ context.Context, start, end time.Time, _ string) ([]LeaderboardModelStat, error) {
	k := f.kind(start, end)
	f.hit("stats:" + k)
	if k == "hist" && f.histGate != nil {
		<-f.histGate
	}
	if k == "prev" && f.prevCheck != nil {
		f.prevCheck(start, end)
	}
	return f.stats[k], nil
}

func (f *fakeLeaderboardRepo) GetMonthActiveUsers(_ context.Context, start, end time.Time) (map[string]int64, error) {
	k := f.kind(start, end)
	f.hit("users:" + k)
	return f.users[k], nil
}

func (f *fakeLeaderboardRepo) GetModelUserPairs(_ context.Context, start, end time.Time, _ string) ([]LeaderboardModelUser, error) {
	k := f.kind(start, end)
	f.hit("pairs:" + k)
	return f.pairs[k], nil
}

func (f *fakeLeaderboardRepo) GetDailyModelUsage(_ context.Context, start, _ time.Time, _ string) ([]LeaderboardBucketPoint, error) {
	k := start.Format("2006-01")
	f.hit("daily:" + k)
	return f.daily[k], nil
}

func lbTestTime(month, day int) time.Time {
	return time.Date(2026, time.Month(month), day, 12, 0, 0, 0, timezone.Location())
}

func pairsOf(m map[string][]int64) []LeaderboardModelUser {
	var out []LeaderboardModelUser
	for model, ids := range m {
		for _, id := range ids {
			out = append(out, LeaderboardModelUser{Model: model, UserID: id})
		}
	}
	return out
}

// 场景：当前 2026-09-15 12:00；历史 = 7、8 月；当月 = 9 月。
func newLeaderboardFixture(t *testing.T) (*fakeLeaderboardRepo, *ModelLeaderboardService, *time.Time) {
	loc := timezone.Location()
	boundary := time.Date(2026, 9, 1, 0, 0, 0, 0, loc)
	repo := &fakeLeaderboardRepo{
		boundary: boundary,
		stats: map[string][]LeaderboardModelStat{
			"hist": {
				{Month: "2026-07", Model: "b", Requests: 600, TotalTokens: 6000, Users: 2, FirstUsedAt: lbTestTime(7, 2), LastUsedAt: lbTestTime(7, 30)},
				{Month: "2026-07", Model: "a", Requests: 5, TotalTokens: 50, Users: 1, DurationMsSum: 1000, DurationCount: 2, FirstUsedAt: lbTestTime(7, 3), LastUsedAt: lbTestTime(7, 4)},
				{Month: "2026-08", Model: "b", Requests: 400, TotalTokens: 4000, Users: 2, Cost: 4, ActualCost: 2, FirstUsedAt: lbTestTime(8, 1), LastUsedAt: lbTestTime(8, 30)},
				{Month: "2026-08", Model: "a", Requests: 5, TotalTokens: 50, Users: 1, FirstUsedAt: lbTestTime(8, 2), LastUsedAt: lbTestTime(8, 3)},
			},
			"2026-09": {
				{Month: "2026-09", Model: "b", Requests: 50, TotalTokens: 500, Cost: 2, ActualCost: 1, FirstUsedAt: lbTestTime(9, 1), LastUsedAt: lbTestTime(9, 15)},
				{Month: "2026-09", Model: "a", Requests: 150, TotalTokens: 900, Cost: 6, ActualCost: 3, DurationMsSum: 2000, DurationCount: 2, FirstUsedAt: lbTestTime(9, 1), LastUsedAt: lbTestTime(9, 14)},
				{Month: "2026-09", Model: "new", Requests: 50, TotalTokens: 600, FirstUsedAt: lbTestTime(9, 10), LastUsedAt: lbTestTime(9, 11)},
			},
			"prev": {
				{Month: "2026-08", Model: "b", Requests: 100},
				{Month: "2026-08", Model: "a", Requests: 100, TotalTokens: 1},
			},
		},
		users: map[string]map[string]int64{
			"hist": {"2026-07": 3, "2026-08": 2},
			"prev": {"2026-08": 2},
		},
		pairs: map[string][]LeaderboardModelUser{
			"hist":    pairsOf(map[string][]int64{"a": {1, 2}, "b": {1, 3}}),
			"2026-09": pairsOf(map[string][]int64{"a": {1, 4}, "b": {3}, "new": {5}}),
		},
		daily: map[string][]LeaderboardBucketPoint{
			"2026-09": {{Bucket: "2026-09-01", Model: "a", Requests: 7}, {Bucket: "2026-09-15", Model: "b", Requests: 3}},
			"2026-08": {{Bucket: "2026-08-03", Model: "b", Requests: 9}},
		},
	}
	now := lbTestTime(9, 15)
	repo.prevCheck = func(start, end time.Time) {
		// 对比期必须止于上月同一进度（如 8/15 12:00），而不是整月
		if want := start.Add(now.Sub(repo.boundary)); !end.Equal(want) {
			t.Errorf("prev period end = %v, want %v", end, want)
		}
	}
	svc := NewModelLeaderboardService(repo)
	svc.now = func() time.Time { return now }
	return repo, svc, &now
}

func TestModelLeaderboard_CurrentMonth(t *testing.T) {
	_, svc, _ := newLeaderboardFixture(t)
	resp, err := svc.Get(context.Background(), "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.IsCurrent || resp.Month != "2026-09" {
		t.Fatalf("month = %s current=%v", resp.Month, resp.IsCurrent)
	}
	r := resp.Monthly.Ranking
	// 50 请求并列时按 tokens 降序：new(600) 在 b(500) 之前
	if len(r) != 3 || r[0].Model != "a" || r[1].Model != "new" || r[2].Model != "b" {
		t.Fatalf("monthly order wrong: %+v", r)
	}
	if r[0].PrevRank == nil || *r[0].PrevRank != 1 || r[0].RequestsGrowth == nil || *r[0].RequestsGrowth != 0.5 {
		t.Fatalf("a growth/prev rank wrong: %+v", r[0])
	}
	if r[1].PrevRank != nil || r[1].RequestsGrowth != nil {
		t.Fatalf("new model should have no prev rank: %+v", r[1])
	}
	if r[2].RequestsGrowth == nil || *r[2].RequestsGrowth != -0.5 {
		t.Fatalf("b growth wrong: %+v", r[2])
	}
	if r[0].RequestShare != 0.6 || r[0].Users != 2 || r[0].AvgDurationMs != 1000 {
		t.Fatalf("a share/users/latency wrong: %+v", r[0])
	}
	s := resp.Monthly.Summary
	if s.Requests != 250 || s.Users != 4 || s.Models != 3 || s.Cost == nil || *s.Cost != 8 {
		t.Fatalf("monthly summary wrong: %+v", s)
	}
	if resp.Monthly.Prev == nil || resp.Monthly.Prev.Requests != 200 || resp.Monthly.Prev.Users != 2 {
		t.Fatalf("prev summary wrong: %+v", resp.Monthly.Prev)
	}
	if len(resp.Monthly.Trend.Labels) != 15 || resp.Monthly.Trend.Labels[14] != "2026-09-15" {
		t.Fatalf("daily labels wrong: %v", resp.Monthly.Trend.Labels)
	}
	if resp.Monthly.Trend.Datasets[0].Model != "a" || resp.Monthly.Trend.Datasets[0].Requests[0] != 7 {
		t.Fatalf("daily dataset wrong: %+v", resp.Monthly.Trend.Datasets)
	}
}

func TestModelLeaderboard_AllTimeMergesHistoryAndLive(t *testing.T) {
	_, svc, _ := newLeaderboardFixture(t)
	resp, err := svc.Get(context.Background(), "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	at := resp.AllTime.Ranking
	if len(at) != 3 || at[0].Model != "b" || at[0].Requests != 1050 || at[1].Model != "a" || at[1].Requests != 160 {
		t.Fatalf("all-time ranking wrong: %+v", at)
	}
	// 对比「截至上月末」：a 仍第 2；new 为新上榜
	if at[1].PrevRank == nil || *at[1].PrevRank != 2 || at[2].PrevRank != nil {
		t.Fatalf("all-time prev rank wrong: %+v", at)
	}
	// 去重用户精确合并：a = {1,2} ∪ {1,4} = 3；b = {1,3} ∪ {3} = 2；全站 = {1..5} = 5
	if at[1].Users != 3 || at[0].Users != 2 || at[2].Users != 1 || resp.AllTime.Summary.Users != 5 {
		t.Fatalf("all-time users wrong: a=%d b=%d new=%d total=%d", at[1].Users, at[0].Users, at[2].Users, resp.AllTime.Summary.Users)
	}
	// 平均耗时按 和/计数 跨月合并：(1000+2000)/(2+2)
	if at[1].AvgDurationMs != 750 {
		t.Fatalf("all-time avg duration = %v, want 750", at[1].AvgDurationMs)
	}
	if resp.AllTime.Summary.FirstAt[:10] != "2026-07-02" || resp.AllTime.Start != resp.AllTime.Summary.FirstAt {
		t.Fatalf("all-time first_at wrong: %+v", resp.AllTime.Summary)
	}
	tr := resp.AllTime.Trend
	if len(tr.Labels) != 3 || tr.Labels[0] != "2026-07" || tr.Labels[2] != "2026-09" || tr.Datasets[0].Model != "b" || tr.Datasets[0].Requests[0] != 600 {
		t.Fatalf("all-time trend wrong: %+v", tr)
	}
	mr := resp.MonthRows
	if len(mr) != 3 || mr[0].Month != "2026-09" || mr[0].TopModel != "a" || mr[0].TopModelShare != 0.6 || mr[0].Users != 4 {
		t.Fatalf("month rows wrong: %+v", mr)
	}
	if mr[1].Month != "2026-08" || mr[1].Users != 2 || mr[1].Models != 2 || *mr[1].Cost != 4 {
		t.Fatalf("august row wrong: %+v", mr[1])
	}
	if len(resp.Months) != 3 || resp.Months[0] != "2026-09" || resp.Months[2] != "2026-07" {
		t.Fatalf("months wrong: %v", resp.Months)
	}
}

func TestModelLeaderboard_CacheLayers(t *testing.T) {
	repo, svc, now := newLeaderboardFixture(t)
	ctx := context.Background()
	if _, err := svc.Get(ctx, "", "", "", true); err != nil {
		t.Fatal(err)
	}
	calls := repo.total()

	// 60s 内：任意月份/身份都不再查库；非管理员费用被抹掉
	user, err := svc.Get(ctx, "requested", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if repo.total() != calls {
		t.Fatalf("expected cache hit, calls %d -> %d", calls, repo.total())
	}
	if user.CostVisible || user.Monthly.Summary.Cost != nil || user.Monthly.Prev.Cost != nil || user.Monthly.Ranking[0].Cost != nil || user.AllTime.Ranking[0].ActualCost != nil || user.MonthRows[0].Cost != nil {
		t.Fatalf("cost not redacted for non-admin")
	}
	if again, _ := svc.Get(ctx, "", "", "", true); again.Monthly.Ranking[0].Cost == nil {
		t.Fatalf("redaction leaked into admin response")
	}

	// 实时层过期：只重查当月，不重算历史快照
	*now = now.Add(61 * time.Second)
	if _, err := svc.Get(ctx, "", "", "", true); err != nil {
		t.Fatal(err)
	}
	if repo.count("stats:2026-09") != 2 || repo.count("pairs:2026-09") != 2 {
		t.Fatalf("live layer not refreshed: %v", repo.calls)
	}
	if repo.count("stats:hist") != 1 || repo.count("pairs:hist") != 1 || repo.count("users:hist") != 1 {
		t.Fatalf("history should stay cached: %v", repo.calls)
	}

	// 历史快照过期（1h）：重建一次
	*now = now.Add(time.Hour)
	if _, err := svc.Get(ctx, "", "", "", true); err != nil {
		t.Fatal(err)
	}
	if repo.count("stats:hist") != 2 {
		t.Fatalf("history should rebuild after TTL: %v", repo.calls)
	}

	// 跨月：边界变化，历史快照立即重建
	*now = time.Date(2026, 10, 1, 0, 0, 30, 0, timezone.Location())
	repo.boundary = time.Date(2026, 10, 1, 0, 0, 0, 0, timezone.Location())
	resp, err := svc.Get(ctx, "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.count("stats:hist") != 3 || resp.Month != "2026-10" {
		t.Fatalf("history should rebuild on month change: %v month=%s", repo.calls, resp.Month)
	}
}

func TestModelLeaderboard_ClosedMonthView(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	ctx := context.Background()
	resp, err := svc.Get(ctx, "", "2026-08", "", true)
	if err != nil {
		t.Fatal(err)
	}
	if resp.IsCurrent || resp.Month != "2026-08" {
		t.Fatalf("closed month flags wrong: %+v", resp)
	}
	r := resp.Monthly.Ranking
	if len(r) != 2 || r[0].Model != "b" || r[0].Requests != 400 {
		t.Fatalf("august ranking wrong: %+v", r)
	}
	// 历史月份整月对比上一整月（7 月 b=600）
	if r[0].RequestsGrowth == nil || *r[0].RequestsGrowth != (400.0-600.0)/600.0 || resp.Monthly.Prev.Users != 3 {
		t.Fatalf("august growth/prev wrong: %+v %+v", r[0], resp.Monthly.Prev)
	}
	if resp.Monthly.Summary.Users != 2 || len(resp.Monthly.Trend.Labels) != 31 || resp.Monthly.Trend.Datasets[0].Requests[2] != 9 {
		t.Fatalf("august summary/trend wrong: %+v", resp.Monthly)
	}
	// 非当月视图：历史累计不做排名对比
	if resp.AllTime.Ranking[0].PrevRank != nil {
		t.Fatalf("all-time should have no comparison for closed month view")
	}
	// 已结束月份的日趋势只查一次
	if _, err := svc.Get(ctx, "", "2026-08", "", false); err != nil {
		t.Fatal(err)
	}
	if repo.count("daily:2026-08") != 1 {
		t.Fatalf("closed month daily should be cached: %v", repo.calls)
	}
}

func TestModelLeaderboard_ConcurrentColdStartSingleFlight(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	repo.histGate = make(chan struct{})
	var wg sync.WaitGroup
	errs := make(chan error, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.Get(context.Background(), "", "", "", i%2 == 0); err != nil {
				errs <- err
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(repo.histGate)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if n := repo.count("stats:hist"); n != 1 {
		t.Fatalf("history built %d times, want 1", n)
	}
	if n := repo.count("stats:2026-09"); n != 1 {
		t.Fatalf("live built %d times, want 1", n)
	}
}

func TestModelLeaderboard_EmptyDatabaseHasNoNullArrays(t *testing.T) {
	repo := &fakeLeaderboardRepo{boundary: time.Date(2026, 9, 1, 0, 0, 0, 0, timezone.Location())}
	svc := NewModelLeaderboardService(repo)
	svc.now = func() time.Time { return lbTestTime(9, 15) }
	resp, err := svc.Get(context.Background(), "", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "null") {
		t.Fatalf("empty response must not contain null arrays: %s", b)
	}
}

// 跨月瞬间：旧月份的慢构建晚于新月份完成时，不能覆盖新缓存，也不能被新请求复用。
func TestModelLeaderboard_RolloverDoesNotReuseStaleBuild(t *testing.T) {
	repo, svc, now := newLeaderboardFixture(t)
	repo.prevCheck = nil
	repo.histGate = make(chan struct{})
	ctx := context.Background()

	oldDone := make(chan error, 1)
	go func() {
		_, err := svc.Get(ctx, "", "", "", true)
		oldDone <- err
	}()
	time.Sleep(30 * time.Millisecond) // 旧月份构建阻塞在历史查询

	*now = time.Date(2026, 10, 1, 0, 0, 5, 0, timezone.Location())
	newDone := make(chan *ModelLeaderboardResponse, 1)
	go func() {
		resp, _ := svc.Get(ctx, "", "", "", true)
		newDone <- resp
	}()
	time.Sleep(30 * time.Millisecond)
	close(repo.histGate)

	if err := <-oldDone; err != nil {
		t.Fatal(err)
	}
	resp := <-newDone
	if resp == nil || resp.Month != "2026-10" || !resp.IsCurrent {
		t.Fatalf("new-month request got stale data: %+v", resp)
	}
	if n := repo.count("stats:hist"); n != 2 {
		t.Fatalf("history should be built once per boundary, got %d", n)
	}
	svc.mu.Lock()
	h := svc.history[usagestats.ModelSourceRequested]
	svc.mu.Unlock()
	if h == nil || !h.boundary.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, timezone.Location())) {
		t.Fatalf("stale build overwrote newer history cache")
	}
}

func TestModelLeaderboard_TokensMetric(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	repo.stats["2026-09"][2].TotalTokens = 5000 // new：调用最少但 Token 最多
	repo.daily["2026-09"] = []LeaderboardBucketPoint{
		{Bucket: "2026-09-01", Model: "a", Requests: 7, TotalTokens: 10},
		{Bucket: "2026-09-15", Model: "b", Requests: 3, TotalTokens: 50},
	}
	ctx := context.Background()

	byReq, err := svc.Get(ctx, "", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	calls := repo.total()
	resp, err := svc.Get(ctx, "", "", LeaderboardMetricTokens, true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.total() != calls {
		t.Fatalf("switching metric must not query again: %d -> %d", calls, repo.total())
	}
	if byReq.Metric != LeaderboardMetricRequests || resp.Metric != LeaderboardMetricTokens {
		t.Fatalf("metric echo wrong: %q %q", byReq.Metric, resp.Metric)
	}

	r := resp.Monthly.Ranking
	if len(r) != 3 || r[0].Model != "new" || r[1].Model != "a" || r[2].Model != "b" || r[0].Rank != 1 {
		t.Fatalf("monthly token order wrong: %+v", r)
	}
	// 对比期按 Token 排：a(1) 第 1，b(0) 第 2
	if r[1].PrevRank == nil || *r[1].PrevRank != 1 || r[1].TokensGrowth == nil || *r[1].TokensGrowth != 899 || *r[1].RequestsGrowth != 0.5 {
		t.Fatalf("a token prev/growth wrong: %+v", r[1])
	}
	if r[2].PrevRank == nil || *r[2].PrevRank != 2 || r[2].PrevTokens == nil || *r[2].PrevTokens != 0 || r[2].TokensGrowth != nil {
		t.Fatalf("b token prev/growth wrong: %+v", r[2])
	}
	if r[0].PrevRank != nil {
		t.Fatalf("new should still be a new entry: %+v", r[0])
	}
	if resp.Monthly.Trend.Datasets[0].Model != "b" {
		t.Fatalf("daily series should be ordered by tokens: %+v", resp.Monthly.Trend.Datasets)
	}

	at := resp.AllTime.Ranking
	if len(at) != 3 || at[0].Model != "b" || at[1].Model != "new" || at[2].Model != "a" || at[2].PrevRank == nil || *at[2].PrevRank != 2 {
		t.Fatalf("all-time token order wrong: %+v", at)
	}
	if mr := resp.MonthRows[0]; mr.TopModel != "new" || mr.TopModelShare != 5000.0/6400.0 {
		t.Fatalf("month row token top wrong: %+v", mr)
	}
	if mr := byReq.MonthRows[0]; mr.TopModel != "a" || mr.TopModelShare != 0.6 {
		t.Fatalf("month row request top wrong: %+v", mr)
	}

	if _, err := svc.Get(ctx, "", "", "cost", true); err == nil {
		t.Fatal("expected invalid metric error")
	}
}

func TestModelLeaderboard_BucketSeriesOther(t *testing.T) {
	points := make([]LeaderboardBucketPoint, 0)
	for i := 0; i < leaderboardTrendTopN+3; i++ {
		points = append(points, LeaderboardBucketPoint{Bucket: "2026-09", Model: string(rune('a' + i)), Requests: int64(100 - i), TotalTokens: 1})
	}
	s := lbBuildBucketSeries(points, []string{"2026-08", "2026-09"}, LeaderboardMetricRequests)
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
