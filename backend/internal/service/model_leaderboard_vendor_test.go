package service

import (
	"context"
	"reflect"
	"testing"
)

func TestLeaderboardVendorOf(t *testing.T) {
	cases := map[string]string{
		"claude-opus-4-7":           LeaderboardVendorClaude,
		"opus":                      LeaderboardVendorClaude,
		"Opus-4.5":                  LeaderboardVendorClaude,
		"anthropic/claude-sonnet-4": LeaderboardVendorClaude,
		"claude-haiku-4-5-20251001": LeaderboardVendorClaude,
		"deepseek-v3.2":             LeaderboardVendorDeepSeek,
		"deepseek-ai/DeepSeek-R1":   LeaderboardVendorDeepSeek,
		"k3":                        LeaderboardVendorMoonshot,
		"K3-256k":                   LeaderboardVendorMoonshot,
		"k2-thinking":               LeaderboardVendorMoonshot,
		"k2.5":                      LeaderboardVendorMoonshot,
		"kimi-k2.5":                 LeaderboardVendorMoonshot,
		"moonshot-v1-8k":            LeaderboardVendorMoonshot,
		"moonshotai/Kimi-K2":        LeaderboardVendorMoonshot,
		"glm-4.6":                   LeaderboardVendorZhipu,
		"chatglm3":                  LeaderboardVendorZhipu,
		"qwen3-coder-plus":          LeaderboardVendorQwen,
		"MiniMax-M2":                LeaderboardVendorMiniMax,
		"grok-4":                    LeaderboardVendorXAI,
		"doubao-seed-1.6":           LeaderboardVendorDoubao,
		"mistral-large":             LeaderboardVendorMistral,
		"llama-3.1-70b":             LeaderboardVendorMeta,
		"gemini-2.5-pro":            LeaderboardVendorGemini,
		"gemma-3-27b":               LeaderboardVendorGemini,
		"gpt-5":                     LeaderboardVendorOpenAI,
		"gpt-5.1-codex":             LeaderboardVendorOpenAI,
		"o3":                        LeaderboardVendorOpenAI,
		"o4-mini":                   LeaderboardVendorOpenAI,
		"openai/gpt-oss-120b":       LeaderboardVendorOpenAI,
		"tts-1":                     LeaderboardVendorOpenAI,
		"text-embedding-3-small":    LeaderboardVendorOpenAI,
		"":                          LeaderboardVendorOther,
		"keye":                      LeaderboardVendorOther,
		"k8s-helper":                LeaderboardVendorOther,
		"okapi":                     LeaderboardVendorOther,
		"some-random-model":         LeaderboardVendorOther,
	}
	for model, want := range cases {
		if got := LeaderboardVendorOf(model); got != want {
			t.Errorf("LeaderboardVendorOf(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestParseLeaderboardVendor(t *testing.T) {
	for _, v := range []string{"", LeaderboardVendorClaude, LeaderboardVendorMoonshot, LeaderboardVendorOther} {
		if got, err := ParseLeaderboardVendor(v); err != nil || got != v {
			t.Errorf("ParseLeaderboardVendor(%q) = %q, %v", v, got, err)
		}
	}
	for _, v := range []string{"Claude", "anthropic", "all", "x"} {
		if _, err := ParseLeaderboardVendor(v); err == nil {
			t.Errorf("ParseLeaderboardVendor(%q) should fail", v)
		}
	}
}

func TestLeaderboardVendorsWithDataOtherLast(t *testing.T) {
	rows := []LeaderboardModelStat{
		{Model: "mystery", Requests: 999, TotalTokens: 1},
		{Model: "gpt-5", Requests: 1, TotalTokens: 50},
		{Model: "opus", Requests: 2, TotalTokens: 10},
	}
	if got := lbVendorsWithData(LeaderboardMetricRequests, lbVendorMemo(), rows); !reflect.DeepEqual(got, []string{"claude", "openai", "other"}) {
		t.Fatalf("requests order = %v", got)
	}
	if got := lbVendorsWithData(LeaderboardMetricTokens, lbVendorMemo(), rows); !reflect.DeepEqual(got, []string{"openai", "claude", "other"}) {
		t.Fatalf("tokens order = %v", got)
	}
}

// 复用标准场景，把 a/b/new 改名为 Claude / DeepSeek / Moonshot 模型。
func newVendorLeaderboardFixture(t *testing.T) (*fakeLeaderboardRepo, *ModelLeaderboardService) {
	repo, svc, _ := newLeaderboardFixture(t)
	rename := map[string]string{"a": "claude-sonnet-4-6", "b": "deepseek-chat", "new": "k3"}
	for _, rows := range repo.stats {
		for i := range rows {
			rows[i].Model = rename[rows[i].Model]
		}
	}
	for _, ps := range repo.pairs {
		for i := range ps {
			ps[i].Model = rename[ps[i].Model]
		}
	}
	for _, pts := range repo.daily {
		for i := range pts {
			pts[i].Model = rename[pts[i].Model]
		}
	}
	return repo, svc
}

func TestModelLeaderboard_VendorFilter(t *testing.T) {
	repo, svc := newVendorLeaderboardFixture(t)
	ctx := context.Background()

	all, err := svc.Get(ctx, LeaderboardQuery{}, true)
	if err != nil {
		t.Fatal(err)
	}
	calls := repo.total()
	if all.Vendor != "" || !reflect.DeepEqual(all.Vendors, []string{"deepseek", "claude", "moonshot"}) {
		t.Fatalf("vendors wrong: %q %v", all.Vendor, all.Vendors)
	}
	if all.Monthly.Summary.Users != 4 || all.AllTime.Summary.Users != 5 {
		t.Fatalf("unfiltered users changed: %+v", all.Monthly.Summary)
	}

	claude, err := svc.Get(ctx, LeaderboardQuery{Vendor: LeaderboardVendorClaude}, true)
	if err != nil {
		t.Fatal(err)
	}
	if repo.total() != calls {
		t.Fatalf("switching vendor must not query again: %d -> %d", calls, repo.total())
	}
	if claude.Vendor != "claude" || !reflect.DeepEqual(claude.Vendors, all.Vendors) {
		t.Fatalf("vendor echo/list wrong: %q %v", claude.Vendor, claude.Vendors)
	}
	r := claude.Monthly.Ranking
	if len(r) != 1 || r[0].Model != "claude-sonnet-4-6" || r[0].RequestShare != 1 || r[0].PrevRank == nil || *r[0].PrevRank != 1 {
		t.Fatalf("claude monthly ranking wrong: %+v", r)
	}
	// 当月 {1,4}；上月同期只有用户 3 用过 Claude
	if s := claude.Monthly.Summary; s.Requests != 150 || s.Users != 2 || s.Models != 1 || claude.Monthly.Prev.Users != 1 || claude.Monthly.Prev.Requests != 100 {
		t.Fatalf("claude summary wrong: %+v prev=%+v", s, claude.Monthly.Prev)
	}
	if ds := claude.Monthly.Trend.Datasets; len(ds) != 1 || ds[0].Model != "claude-sonnet-4-6" {
		t.Fatalf("claude daily trend wrong: %+v", ds)
	}
	// 历史累计：{1,2} ∪ {1,4} = 3
	if at := claude.AllTime; len(at.Ranking) != 1 || at.Ranking[0].Requests != 160 || at.Summary.Users != 3 || at.Ranking[0].Users != 3 {
		t.Fatalf("claude all-time wrong: %+v", at)
	}
	if mr := claude.MonthRows; len(mr) != 3 || mr[0].Users != 2 || mr[1].Users != 1 || mr[2].Users != 1 || mr[1].TopModel != "claude-sonnet-4-6" || mr[1].TopModelShare != 1 {
		t.Fatalf("claude month rows wrong: %+v", mr)
	}

	// 只在当月出现的供应商：月度明细不补空月，但月份选择器仍是全部月份
	kimi, err := svc.Get(ctx, LeaderboardQuery{Vendor: LeaderboardVendorMoonshot}, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(kimi.MonthRows) != 1 || kimi.MonthRows[0].Month != "2026-09" || len(kimi.Months) != 3 {
		t.Fatalf("moonshot month rows/months wrong: %+v %v", kimi.MonthRows, kimi.Months)
	}
	if r := kimi.Monthly.Ranking; len(r) != 1 || r[0].Model != "k3" || r[0].PrevRank != nil || kimi.Monthly.Prev.Users != 0 {
		t.Fatalf("moonshot should be a new entry: %+v", r)
	}

	// 已结束月份：8 月 DeepSeek 用户 {1,3}，对比 7 月 {1,3}
	ds, err := svc.Get(ctx, LeaderboardQuery{Month: "2026-08", Vendor: LeaderboardVendorDeepSeek}, true)
	if err != nil {
		t.Fatal(err)
	}
	if ds.Monthly.Summary.Users != 2 || ds.Monthly.Prev.Users != 2 || ds.Monthly.Summary.Requests != 400 || len(ds.Monthly.Ranking) != 1 {
		t.Fatalf("deepseek august wrong: %+v prev=%+v", ds.Monthly.Summary, ds.Monthly.Prev)
	}

	// 没有数据的供应商：合法但为空
	empty, err := svc.Get(ctx, LeaderboardQuery{Vendor: LeaderboardVendorGemini}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Monthly.Ranking) != 0 || len(empty.MonthRows) != 0 || empty.AllTime.Summary.Users != 0 || empty.Monthly.Trend.Datasets == nil {
		t.Fatalf("empty vendor wrong: %+v", empty)
	}

	if _, err := svc.Get(ctx, LeaderboardQuery{Vendor: "anthropic"}, true); err == nil {
		t.Fatal("expected invalid vendor error")
	}
}
