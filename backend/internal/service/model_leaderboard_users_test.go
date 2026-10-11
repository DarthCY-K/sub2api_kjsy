package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// 用户 × 模型 用量：7、8 月为历史快照，9 月为当月。
func withUserUsage(repo *fakeLeaderboardRepo) {
	u := func(month, model string, user, req, tok int64, cost float64, day int) LeaderboardModelUser {
		m := 7
		switch month {
		case "2026-08":
			m = 8
		case "2026-09":
			m = 9
		}
		return LeaderboardModelUser{
			Month: month, Model: model, UserID: user, Requests: req, TotalTokens: tok,
			InputTokens: tok / 2, OutputTokens: tok / 2, Cost: cost, ActualCost: cost / 2, LastUsedAt: lbTestTime(m, day),
		}
	}
	repo.pairs = map[string][]LeaderboardModelUser{
		"hist": {
			u("2026-07", "gpt-5", 1, 100, 1000, 1, 5),
			u("2026-07", "claude-sonnet-4", 2, 5, 50, 0, 6),
			u("2026-08", "gpt-5", 1, 50, 500, 1, 7),
			u("2026-08", "gpt-5", 3, 300, 300, 3, 8),
			u("2026-08", "claude-sonnet-4", 1, 10, 100, 0, 9),
		},
		"2026-09": {
			u("2026-09", "claude-sonnet-4", 1, 20, 2000, 2, 10),
			u("2026-09", "claude-sonnet-4", 4, 100, 100, 1, 11),
			u("2026-09", "gpt-5", 3, 1, 10, 0, 12),
			u("2026-09", "deepseek-v3", 5, 30, 30, 0, 13),
		},
	}
	repo.idents = map[int64]LeaderboardUserIdentity{
		1: {UserID: 1, Username: "darthcy", Email: "darth@example.com"},
		3: {UserID: 3, Email: "abcdef@gmail.com"},
		4: {UserID: 4, Username: "李四", Email: "li4@example.com", Deleted: true},
		5: {UserID: 5, Username: "bob", Email: "bob@example.com"},
		// 2 不存在（已被物理删除）
	}
}

func userIDs(items []LeaderboardUserItem) []int64 {
	out := make([]int64, 0, len(items))
	for _, it := range items {
		out = append(out, it.UserID)
	}
	return out
}

func TestModelLeaderboard_UserRankingAdmin(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	withUserUsage(repo)
	resp, err := svc.Get(context.Background(), LeaderboardQuery{ViewerID: 1}, true)
	if err != nil {
		t.Fatal(err)
	}

	// 当月按请求数：4(100) > 5(30) > 1(20) > 3(1)
	m := resp.Monthly.UserRanking
	if got := userIDs(m); len(got) != 4 || got[0] != 4 || got[1] != 5 || got[2] != 1 || got[3] != 3 {
		t.Fatalf("monthly user order wrong: %v", got)
	}
	if m[0].Rank != 1 || m[0].Name != "李四" || m[0].Email != "li4@example.com" || !m[0].Deleted || m[0].RequestShare != 100.0/151.0 {
		t.Fatalf("admin top user wrong: %+v", m[0])
	}
	if m[0].Cost == nil || *m[0].Cost != 1 || m[0].ActualCost == nil || *m[0].ActualCost != 0.5 {
		t.Fatalf("admin should see cost: %+v", m[0])
	}
	if !m[2].IsSelf || m[2].Name != "darthcy" || m[2].UserID != 1 {
		t.Fatalf("self row wrong: %+v", m[2])
	}
	if resp.Monthly.UserSelf != nil {
		t.Fatalf("self is in top N, user_self must be nil")
	}

	// 历史累计：1 = 100+50+10+20 = 180，3 = 301，4 = 100，5 = 30，2 = 5
	a := resp.AllTime.UserRanking
	if got := userIDs(a); len(got) != 5 || got[0] != 3 || got[1] != 1 || got[2] != 4 || got[3] != 5 || got[4] != 2 {
		t.Fatalf("all-time user order wrong: %v", got)
	}
	u1 := a[1]
	if u1.Requests != 180 || u1.TotalTokens != 3600 || u1.InputTokens != 1800 || *u1.Cost != 4 || u1.LastUsedAt[:10] != "2026-09-10" {
		t.Fatalf("user 1 all-time totals wrong: %+v", u1)
	}
	if len(u1.Models) != 2 || u1.Models[0].Model != "gpt-5" || u1.Models[0].Requests != 150 || u1.Models[0].Share != 150.0/180.0 ||
		u1.Models[1].Model != "claude-sonnet-4" || u1.Models[1].Requests != 30 || *u1.Models[0].Cost != 2 {
		t.Fatalf("user 1 model breakdown wrong: %+v", u1.Models)
	}
	if a[0].Name != "abcdef@gmail.com" || a[4].Name != "" || a[4].UserID != 2 {
		t.Fatalf("admin names wrong: %q %q", a[0].Name, a[4].Name)
	}

	// 只查一次身份，且两个口径的 ID 合并去重
	if repo.count("idents") != 1 || len(repo.identQueried) != 5 {
		t.Fatalf("identity lookups wrong: calls=%d ids=%v", repo.count("idents"), repo.identQueried)
	}
}

func TestModelLeaderboard_UserRankingNonAdminMasked(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	withUserUsage(repo)
	resp, err := svc.Get(context.Background(), LeaderboardQuery{ViewerID: 1}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []LeaderboardPeriod{resp.Monthly, resp.AllTime} {
		for _, it := range p.UserRanking {
			if it.UserID != 0 || it.Email != "" || it.Cost != nil || it.ActualCost != nil {
				t.Fatalf("non-admin leaked identity/cost: %+v", it)
			}
			if (it.LastUsedAt == "") != !it.IsSelf {
				t.Fatalf("last_used_at should be hidden for others only: %+v", it)
			}
			for _, m := range it.Models {
				if m.Cost != nil || m.ActualCost != nil {
					t.Fatalf("non-admin leaked model cost: %+v", m)
				}
			}
		}
	}
	m := resp.Monthly.UserRanking
	if m[0].Name != "李*" || !m[0].Deleted || m[1].Name != "b***" {
		t.Fatalf("masked names wrong: %q %q", m[0].Name, m[1].Name)
	}
	if !m[2].IsSelf || m[2].Name != "darthcy" {
		t.Fatalf("self should see own real name: %+v", m[2])
	}
	if a := resp.AllTime.UserRanking; a[0].Name != "ab***@gmail.com" || a[4].Name != "" {
		t.Fatalf("all-time masked names wrong: %q %q", a[0].Name, a[4].Name)
	}

	b, err := json.Marshal(resp.Monthly.UserRanking)
	if err != nil {
		t.Fatal(err)
	}
	for _, leak := range []string{"user_id", "email", "li4@", "darth@", "cost"} {
		if strings.Contains(string(b), leak) {
			t.Fatalf("non-admin JSON contains %q: %s", leak, b)
		}
	}
}

func TestModelLeaderboard_UserRankingClosedMonthAndMetric(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	withUserUsage(repo)
	ctx := context.Background()

	aug, err := svc.Get(ctx, LeaderboardQuery{Month: "2026-08"}, true)
	if err != nil {
		t.Fatal(err)
	}
	// 8 月：3 = 300，1 = 50+10 = 60
	if got := userIDs(aug.Monthly.UserRanking); len(got) != 2 || got[0] != 3 || got[1] != 1 || aug.Monthly.UserRanking[1].Requests != 60 {
		t.Fatalf("august user ranking wrong: %+v", aug.Monthly.UserRanking)
	}

	calls := repo.total()
	tok, err := svc.Get(ctx, LeaderboardQuery{Metric: LeaderboardMetricTokens}, true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.total() != calls {
		t.Fatalf("switching metric must not query again: %d -> %d", calls, repo.total())
	}
	// 历史累计按 Token：1 = 3600 居首；其模型按 Token 排：claude 2100 > gpt 1500
	a := tok.AllTime.UserRanking
	if a[0].UserID != 1 || a[0].Models[0].Model != "claude-sonnet-4" || a[0].Models[0].Share != 2100.0/3600.0 {
		t.Fatalf("token ranking wrong: %+v", a[0])
	}
}

func TestModelLeaderboard_UserRankingVendorFilter(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	withUserUsage(repo)
	resp, err := svc.Get(context.Background(), LeaderboardQuery{Vendor: "claude"}, true)
	if err != nil {
		t.Fatal(err)
	}
	// 只统计 claude：1 = 10+20 = 30，4 = 100，2 = 5；3、5 无 claude 调用
	a := resp.AllTime.UserRanking
	if got := userIDs(a); len(got) != 3 || got[0] != 4 || got[1] != 1 || got[2] != 2 {
		t.Fatalf("vendor-filtered users wrong: %v", got)
	}
	if a[1].Requests != 30 || len(a[1].Models) != 1 || a[1].Models[0].Model != "claude-sonnet-4" {
		t.Fatalf("vendor-filtered user 1 wrong: %+v", a[1])
	}
}

func TestModelLeaderboard_UserSelfOutsideTopN(t *testing.T) {
	repo, svc, _ := newLeaderboardFixture(t)
	rows := make([]LeaderboardModelUser, 0, leaderboardUserTopN+5)
	for i := 1; i <= leaderboardUserTopN+5; i++ {
		rows = append(rows, LeaderboardModelUser{Month: "2026-09", Model: "gpt-5", UserID: int64(i), Requests: int64(1000 - i)})
	}
	repo.pairs = map[string][]LeaderboardModelUser{"2026-09": rows}
	viewer := int64(leaderboardUserTopN + 3)
	resp, err := svc.Get(context.Background(), LeaderboardQuery{ViewerID: viewer}, false)
	if err != nil {
		t.Fatal(err)
	}
	m := resp.Monthly
	if len(m.UserRanking) != leaderboardUserTopN || m.UserRanking[leaderboardUserTopN-1].Rank != leaderboardUserTopN {
		t.Fatalf("top N wrong: %d", len(m.UserRanking))
	}
	if m.UserSelf == nil || m.UserSelf.Rank != int(viewer) || !m.UserSelf.IsSelf || m.UserSelf.Requests != 1000-viewer || len(m.UserSelf.Models) != 1 {
		t.Fatalf("user_self wrong: %+v", m.UserSelf)
	}
	for _, it := range m.UserRanking {
		if it.IsSelf {
			t.Fatalf("self must not be in top N: %+v", it)
		}
	}
	// 只为上榜用户 + 本人查询身份
	if len(repo.identQueried) != leaderboardUserTopN+1 {
		t.Fatalf("identity lookup should cover top N + self, got %d ids", len(repo.identQueried))
	}
}

func TestModelLeaderboard_UserIdentityCache(t *testing.T) {
	repo, svc, now := newLeaderboardFixture(t)
	repo.prevCheck = nil
	withUserUsage(repo)
	ctx := context.Background()
	if _, err := svc.Get(ctx, LeaderboardQuery{}, true); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(ctx, LeaderboardQuery{}, false); err != nil {
		t.Fatal(err)
	}
	if repo.count("idents") != 1 {
		t.Fatalf("identities should be cached: %d", repo.count("idents"))
	}

	repo.idents[1] = LeaderboardUserIdentity{UserID: 1, Username: "renamed"}
	*now = now.Add(leaderboardIdentityTTL + time.Second)
	resp, err := svc.Get(ctx, LeaderboardQuery{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.count("idents") != 2 {
		t.Fatalf("identities should refresh after TTL: %d", repo.count("idents"))
	}
	if resp.AllTime.UserRanking[1].Name != "renamed" {
		t.Fatalf("renamed user not refreshed: %+v", resp.AllTime.UserRanking[1])
	}
}

func TestLbMaskIdentity(t *testing.T) {
	cases := []struct {
		in   LeaderboardUserIdentity
		want string
	}{
		{LeaderboardUserIdentity{Username: "darthcy", Email: "x@y.com"}, "da***y"},
		{LeaderboardUserIdentity{Username: "bob"}, "b***"},
		{LeaderboardUserIdentity{Username: "ab"}, "a*"},
		{LeaderboardUserIdentity{Username: "z"}, "*"},
		{LeaderboardUserIdentity{Username: "张三丰大侠"}, "张三***侠"},
		{LeaderboardUserIdentity{Username: "  ", Email: "abcdef@gmail.com"}, "ab***@gmail.com"},
		{LeaderboardUserIdentity{Email: "ab@qq.com"}, "a***@qq.com"},
		{LeaderboardUserIdentity{Username: "someone@corp.io"}, "so***@corp.io"},
		{LeaderboardUserIdentity{}, ""},
	}
	for _, c := range cases {
		if got := lbMaskIdentity(c.in); got != c.want {
			t.Errorf("lbMaskIdentity(%+v) = %q, want %q", c.in, got, c.want)
		}
	}
}
