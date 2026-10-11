package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

const (
	leaderboardUserTopN    = 100
	leaderboardIdentityTTL = 5 * time.Minute
)

type LeaderboardUserIdentity struct {
	UserID   int64
	Email    string
	Username string
	Deleted  bool
}

// LeaderboardUserModel 用户在单个模型上的用量；Share 为该模型占此用户当前 metric 的比例。
type LeaderboardUserModel struct {
	Model       string   `json:"model"`
	Requests    int64    `json:"requests"`
	TotalTokens int64    `json:"total_tokens"`
	Share       float64  `json:"share"`
	Cost        *float64 `json:"cost,omitempty"`
	ActualCost  *float64 `json:"actual_cost,omitempty"`
}

// LeaderboardUserItem 用户用量排行的一行。
// 非管理员：他人 Name 为脱敏名，UserID/Email 不返回；本人 Name 为真名。
type LeaderboardUserItem struct {
	Rank                int                    `json:"rank"`
	UserID              int64                  `json:"user_id,omitempty"` // 仅管理员
	Name                string                 `json:"name"`
	Email               string                 `json:"email,omitempty"` // 仅管理员
	Deleted             bool                   `json:"deleted"`
	IsSelf              bool                   `json:"is_self"`
	Requests            int64                  `json:"requests"`
	RequestShare        float64                `json:"request_share"`
	TotalTokens         int64                  `json:"total_tokens"`
	TokenShare          float64                `json:"token_share"`
	InputTokens         int64                  `json:"input_tokens"`
	OutputTokens        int64                  `json:"output_tokens"`
	CacheCreationTokens int64                  `json:"cache_creation_tokens"`
	CacheReadTokens     int64                  `json:"cache_read_tokens"`
	Models              []LeaderboardUserModel `json:"models"` // 按当前 metric 降序
	LastUsedAt          string                 `json:"last_used_at"`
	Cost                *float64               `json:"cost,omitempty"`        // 仅管理员
	ActualCost          *float64               `json:"actual_cost,omitempty"` // 仅管理员
}

type lbIdentityEntry struct {
	ident LeaderboardUserIdentity
	at    time.Time
}

func lbAddUserStat(dst *LeaderboardModelUser, r LeaderboardModelUser) {
	dst.Requests += r.Requests
	dst.InputTokens += r.InputTokens
	dst.OutputTokens += r.OutputTokens
	dst.CacheCreationTokens += r.CacheCreationTokens
	dst.CacheReadTokens += r.CacheReadTokens
	dst.TotalTokens += r.TotalTokens
	dst.Cost += r.Cost
	dst.ActualCost += r.ActualCost
	if r.LastUsedAt.After(dst.LastUsedAt) {
		dst.LastUsedAt = r.LastUsedAt
	}
}

// lbMergeUserStats 按 (用户, 模型) 合并多组统计（返回新切片，不修改入参）。
func lbMergeUserStats(groups ...[]LeaderboardModelUser) []LeaderboardModelUser {
	type key struct {
		user  int64
		model string
	}
	idx := map[key]int{}
	out := make([]LeaderboardModelUser, 0, 256)
	for _, g := range groups {
		for _, r := range g {
			k := key{r.UserID, r.Model}
			if i, ok := idx[k]; ok {
				lbAddUserStat(&out[i], r)
				continue
			}
			r.Month = ""
			idx[k] = len(out)
			out = append(out, r)
		}
	}
	return out
}

func lbUserMetricKeys(r LeaderboardModelUser, metric string) (int64, int64) {
	if metric == LeaderboardMetricTokens {
		return r.TotalTokens, r.Requests
	}
	return r.Requests, r.TotalTokens
}

// buildUserRanking 汇总每个用户在 keep 模型上的用量，返回按 metric 的前 N 名，以及不在前 N 名时的 viewer 本人。
// 先只按用户累计总量排序，再仅为上榜用户收集分模型明细，避免为全部用户建立模型索引。
func buildUserRanking(groups [][]LeaderboardModelUser, keep func(string) bool, metric string, viewerID int64, loc *time.Location) ([]LeaderboardUserItem, *LeaderboardUserItem) {
	totals := map[int64]*LeaderboardModelUser{}
	var totalReq, totalTok int64
	for _, g := range groups {
		for _, r := range g {
			if !keep(r.Model) {
				continue
			}
			t := totals[r.UserID]
			if t == nil {
				t = &LeaderboardModelUser{UserID: r.UserID}
				totals[r.UserID] = t
			}
			lbAddUserStat(t, r)
			totalReq += r.Requests
			totalTok += r.TotalTokens
		}
	}
	order := make([]*LeaderboardModelUser, 0, len(totals))
	for _, t := range totals {
		order = append(order, t)
	}
	sort.Slice(order, func(i, j int) bool {
		pi, si := lbUserMetricKeys(*order[i], metric)
		pj, sj := lbUserMetricKeys(*order[j], metric)
		if pi != pj {
			return pi > pj
		}
		if si != sj {
			return si > sj
		}
		return order[i].UserID < order[j].UserID
	})

	ranks := make(map[int64]int, leaderboardUserTopN+1)
	for i, t := range order {
		if i < leaderboardUserTopN || t.UserID == viewerID {
			ranks[t.UserID] = i + 1
		}
	}
	models := make(map[int64]map[string]*LeaderboardModelUser, len(ranks))
	for _, g := range groups {
		for _, r := range g {
			if _, ok := ranks[r.UserID]; !ok || !keep(r.Model) {
				continue
			}
			byModel := models[r.UserID]
			if byModel == nil {
				byModel = map[string]*LeaderboardModelUser{}
				models[r.UserID] = byModel
			}
			m := byModel[r.Model]
			if m == nil {
				m = &LeaderboardModelUser{UserID: r.UserID, Model: r.Model}
				byModel[r.Model] = m
			}
			lbAddUserStat(m, r)
		}
	}

	build := func(t *LeaderboardModelUser) LeaderboardUserItem {
		cost, actual := t.Cost, t.ActualCost
		item := LeaderboardUserItem{
			Rank:                ranks[t.UserID],
			UserID:              t.UserID,
			IsSelf:              viewerID != 0 && t.UserID == viewerID,
			Requests:            t.Requests,
			RequestShare:        lbSafeRatio(t.Requests, totalReq),
			TotalTokens:         t.TotalTokens,
			TokenShare:          lbSafeRatio(t.TotalTokens, totalTok),
			InputTokens:         t.InputTokens,
			OutputTokens:        t.OutputTokens,
			CacheCreationTokens: t.CacheCreationTokens,
			CacheReadTokens:     t.CacheReadTokens,
			LastUsedAt:          lbFormatTime(t.LastUsedAt, loc),
			Cost:                &cost,
			ActualCost:          &actual,
		}
		userMetric, _ := lbUserMetricKeys(*t, metric)
		ms := make([]*LeaderboardModelUser, 0, len(models[t.UserID]))
		for _, m := range models[t.UserID] {
			ms = append(ms, m)
		}
		sort.Slice(ms, func(i, j int) bool {
			pi, si := lbUserMetricKeys(*ms[i], metric)
			pj, sj := lbUserMetricKeys(*ms[j], metric)
			if pi != pj {
				return pi > pj
			}
			if si != sj {
				return si > sj
			}
			return ms[i].Model < ms[j].Model
		})
		item.Models = make([]LeaderboardUserModel, 0, len(ms))
		for _, m := range ms {
			mc, ma := m.Cost, m.ActualCost
			v, _ := lbUserMetricKeys(*m, metric)
			item.Models = append(item.Models, LeaderboardUserModel{
				Model:       m.Model,
				Requests:    m.Requests,
				TotalTokens: m.TotalTokens,
				Share:       lbSafeRatio(v, userMetric),
				Cost:        &mc,
				ActualCost:  &ma,
			})
		}
		return item
	}

	n := min(len(order), leaderboardUserTopN)
	top := make([]LeaderboardUserItem, 0, n)
	for _, t := range order[:n] {
		top = append(top, build(t))
	}
	var self *LeaderboardUserItem
	if viewerID != 0 {
		if t := totals[viewerID]; t != nil && ranks[viewerID] > leaderboardUserTopN {
			item := build(t)
			self = &item
		}
	}
	return top, self
}

// lookupUserIdentities 只查缓存中缺失或过期的 ID；查询失败时降级为已有（可能过期）的缓存，不影响排行榜本身。
func (s *ModelLeaderboardService) lookupUserIdentities(ctx context.Context, ids []int64) map[int64]LeaderboardUserIdentity {
	now := s.now()
	out := make(map[int64]LeaderboardUserIdentity, len(ids))
	var missing []int64
	s.identMu.Lock()
	for _, id := range ids {
		e, ok := s.idents[id]
		if ok {
			out[id] = e.ident
		}
		if !ok || now.Sub(e.at) >= leaderboardIdentityTTL {
			missing = append(missing, id)
		}
	}
	s.identMu.Unlock()
	if len(missing) == 0 {
		return out
	}

	found, err := s.repo.GetUserIdentities(ctx, missing)
	if err != nil {
		logger.LegacyPrintf("service.model_leaderboard", "[ModelLeaderboard] 查询用户身份失败: %v", err)
		return out
	}
	s.identMu.Lock()
	defer s.identMu.Unlock()
	for _, id := range missing {
		// 查不到的 ID 也缓存空身份，避免每次请求重复查询
		s.idents[id] = lbIdentityEntry{ident: LeaderboardUserIdentity{UserID: id}, at: now}
	}
	for _, u := range found {
		s.idents[u.UserID] = lbIdentityEntry{ident: u, at: now}
	}
	for _, id := range missing {
		out[id] = s.idents[id].ident
	}
	return out
}

// fillUserIdentities 为用户排行填充名称，并按身份抹掉他人的可识别信息。
func (s *ModelLeaderboardService) fillUserIdentities(ctx context.Context, resp *ModelLeaderboardResponse, isAdmin bool) {
	var items []*LeaderboardUserItem
	for _, p := range []*LeaderboardPeriod{&resp.Monthly, &resp.AllTime} {
		for i := range p.UserRanking {
			items = append(items, &p.UserRanking[i])
		}
		if p.UserSelf != nil {
			items = append(items, p.UserSelf)
		}
	}
	if len(items) == 0 {
		return
	}
	seen := make(map[int64]struct{}, len(items))
	ids := make([]int64, 0, len(items))
	for _, it := range items {
		if _, ok := seen[it.UserID]; !ok {
			seen[it.UserID] = struct{}{}
			ids = append(ids, it.UserID)
		}
	}
	idents := s.lookupUserIdentities(ctx, ids)
	for _, it := range items {
		ident := idents[it.UserID]
		it.Deleted = ident.Deleted
		switch {
		case isAdmin:
			it.Name = lbDisplayName(ident)
			it.Email = ident.Email
		case it.IsSelf:
			it.Name = lbDisplayName(ident)
			it.UserID = 0
		default:
			it.Name = lbMaskIdentity(ident)
			it.UserID = 0
			it.LastUsedAt = ""
		}
	}
}

func lbDisplayName(u LeaderboardUserIdentity) string {
	if name := strings.TrimSpace(u.Username); name != "" {
		return name
	}
	return u.Email
}

// lbMaskIdentity 脱敏展示名：优先用户名，否则邮箱（保留域名）。
func lbMaskIdentity(u LeaderboardUserIdentity) string {
	name := strings.TrimSpace(u.Username)
	if name == "" {
		name = strings.TrimSpace(u.Email)
	}
	if local, domain, ok := strings.Cut(name, "@"); ok {
		r := []rune(local)
		keep := 2
		if len(r) <= 2 {
			keep = 1
		}
		if len(r) == 0 {
			keep = 0
		}
		return string(r[:keep]) + "***@" + domain
	}
	return lbMaskName(name)
}

func lbMaskName(s string) string {
	r := []rune(s)
	switch n := len(r); {
	case n == 0:
		return ""
	case n == 1:
		return "*"
	case n == 2:
		return string(r[0]) + "*"
	case n <= 4:
		return string(r[0]) + "***"
	default:
		return string(r[:2]) + "***" + string(r[n-1])
	}
}
