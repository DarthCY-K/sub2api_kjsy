package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// ModelLeaderboardRepository 模型调用量排行榜聚合查询（只读），区间均为 [start, end)。
type ModelLeaderboardRepository interface {
	// GetMonthModelStats 按 自然月 × 模型 聚合；Users 为该月该模型的去重用户数。
	GetMonthModelStats(ctx context.Context, start, end time.Time, source string) ([]LeaderboardModelStat, error)
	// GetMonthActiveUsers 每个自然月的去重活跃用户数（与模型口径无关）。
	GetMonthActiveUsers(ctx context.Context, start, end time.Time) (map[string]int64, error)
	// GetModelUserPairs 去重的 (模型, 用户) 对，用于跨区间精确合并去重用户数。
	GetModelUserPairs(ctx context.Context, start, end time.Time, source string) ([]LeaderboardModelUser, error)
	// GetDailyModelUsage 按 自然日 × 模型 聚合。
	GetDailyModelUsage(ctx context.Context, start, end time.Time, source string) ([]LeaderboardBucketPoint, error)
}

// LeaderboardModelStat 单模型在一个自然月（或合并后的任意区间）内的可加聚合。
// 耗时类保存 和/计数 而非均值，才能跨月正确合并。
type LeaderboardModelStat struct {
	Month               string // YYYY-MM；跨月合并后为空
	Model               string
	Requests            int64
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	TotalTokens         int64
	Users               int64
	Images              int64
	DurationMsSum       int64
	DurationCount       int64
	FirstTokenMsSum     int64
	FirstTokenCount     int64
	Cost                float64
	ActualCost          float64
	FirstUsedAt         time.Time
	LastUsedAt          time.Time
}

type LeaderboardModelUser struct {
	Model  string
	UserID int64
}

// LeaderboardBucketPoint 时间桶 × 模型。
type LeaderboardBucketPoint struct {
	Bucket      string
	Model       string
	Requests    int64
	TotalTokens int64
}

// ---------- 对外视图（handler 直接序列化） ----------

type LeaderboardRankItem struct {
	Rank                int      `json:"rank"`
	Model               string   `json:"model"`
	Requests            int64    `json:"requests"`
	RequestShare        float64  `json:"request_share"`
	TotalTokens         int64    `json:"total_tokens"`
	TokenShare          float64  `json:"token_share"`
	InputTokens         int64    `json:"input_tokens"`
	OutputTokens        int64    `json:"output_tokens"`
	CacheCreationTokens int64    `json:"cache_creation_tokens"`
	CacheReadTokens     int64    `json:"cache_read_tokens"`
	CacheHitRate        float64  `json:"cache_hit_rate"`
	Users               int64    `json:"users"`
	Images              int64    `json:"images"`
	AvgDurationMs       float64  `json:"avg_duration_ms"`
	AvgFirstTokenMs     float64  `json:"avg_first_token_ms"`
	FirstUsedAt         string   `json:"first_used_at"`
	LastUsedAt          string   `json:"last_used_at"`
	PrevRank            *int     `json:"prev_rank"`             // 对比期排名（按当前 metric）；nil = 新上榜
	PrevRequests        *int64   `json:"prev_requests"`         // 对比期调用量
	RequestsGrowth      *float64 `json:"requests_growth"`       // 环比（小数，0.25=+25%）；对比期无数据为 nil
	PrevTokens          *int64   `json:"prev_tokens"`           // 对比期 Token 量
	TokensGrowth        *float64 `json:"tokens_growth"`         // Token 环比；对比期为 0 或无数据为 nil
	Cost                *float64 `json:"cost,omitempty"`        // 仅管理员
	ActualCost          *float64 `json:"actual_cost,omitempty"` // 仅管理员
}

type LeaderboardSummary struct {
	Requests        int64    `json:"requests"`
	TotalTokens     int64    `json:"total_tokens"`
	InputTokens     int64    `json:"input_tokens"`
	OutputTokens    int64    `json:"output_tokens"`
	CacheReadTokens int64    `json:"cache_read_tokens"`
	Users           int64    `json:"users"`
	Models          int64    `json:"models"`
	FirstAt         string   `json:"first_at,omitempty"`
	LastAt          string   `json:"last_at,omitempty"`
	Cost            *float64 `json:"cost,omitempty"`
	ActualCost      *float64 `json:"actual_cost,omitempty"`
}

// LeaderboardSeries 堆叠/折线图数据：labels × 每模型一条序列（Top N + 其他）。
type LeaderboardSeries struct {
	Labels   []string                  `json:"labels"`
	Datasets []LeaderboardSeriesDetail `json:"datasets"`
}

type LeaderboardSeriesDetail struct {
	Model    string  `json:"model"`
	Requests []int64 `json:"requests"`
	Tokens   []int64 `json:"tokens"`
}

type LeaderboardMonthRow struct {
	Month         string   `json:"month"`
	Requests      int64    `json:"requests"`
	TotalTokens   int64    `json:"total_tokens"`
	Users         int64    `json:"users"`
	Models        int64    `json:"models"`
	TopModel      string   `json:"top_model"`       // 按当前 metric 的第一名
	TopModelShare float64  `json:"top_model_share"` // 第一名在当前 metric 下的占比
	Cost          *float64 `json:"cost,omitempty"`
	ActualCost    *float64 `json:"actual_cost,omitempty"`
}

// LeaderboardPeriod 一个统计口径（当月 / 历史累计）的完整数据。
type LeaderboardPeriod struct {
	Label   string                `json:"label"`
	Start   string                `json:"start"`
	End     string                `json:"end"`
	Summary LeaderboardSummary    `json:"summary"`
	Prev    *LeaderboardSummary   `json:"prev,omitempty"` // 当月口径：上月同期对比
	Ranking []LeaderboardRankItem `json:"ranking"`
	Trend   LeaderboardSeries     `json:"trend"` // 当月=按日，历史=按月
}

type ModelLeaderboardResponse struct {
	Source      string                `json:"source"`
	Metric      string                `json:"metric"` // 排名口径：requests | tokens
	Timezone    string                `json:"timezone"`
	GeneratedAt string                `json:"generated_at"`
	Month       string                `json:"month"`
	IsCurrent   bool                  `json:"is_current"` // month 是否为当前自然月（对比期为上月同期）
	Months      []string              `json:"months"`     // 有数据的月份（倒序），供月份切换
	CostVisible bool                  `json:"cost_visible"`
	Monthly     LeaderboardPeriod     `json:"monthly"`
	AllTime     LeaderboardPeriod     `json:"all_time"`
	MonthRows   []LeaderboardMonthRow `json:"month_rows"`
}

// ---------- service ----------

const (
	leaderboardLiveTTL    = 60 * time.Second
	leaderboardHistoryTTL = time.Hour
	leaderboardQueryLimit = 2 * time.Minute
	leaderboardTrendTopN  = 8
	leaderboardOther      = "__other__"

	LeaderboardMetricRequests = "requests"
	LeaderboardMetricTokens   = "tokens"
)

// ParseLeaderboardMetric 空串 = requests。
func ParseLeaderboardMetric(metric string) (string, error) {
	switch metric {
	case "", LeaderboardMetricRequests:
		return LeaderboardMetricRequests, nil
	case LeaderboardMetricTokens:
		return LeaderboardMetricTokens, nil
	}
	return "", fmt.Errorf("invalid metric %q, want requests|tokens", metric)
}

// lbHistory 已结束月份（created_at < boundary）的聚合快照。
// 这部分数据基本不再变化：按小时刷新，跨月（boundary 变化）立即重建。构建后只读。
type lbHistory struct {
	boundary   time.Time
	builtAt    time.Time
	months     []string // 升序
	byMonth    map[string][]LeaderboardModelStat
	monthUsers map[string]int64
	modelUsers map[string]map[int64]struct{}
	allUsers   map[int64]struct{}
	merged     []LeaderboardModelStat // 全部已结束月份按模型合并，Users 为精确去重值

	dailyMu sync.Mutex
	daily   map[string][]LeaderboardBucketPoint // 已结束月份的日趋势，按需加载
}

// lbLive 当前自然月的实时聚合，构建后只读。
type lbLive struct {
	builtAt    time.Time
	monthStart time.Time
	month      string
	merged     []LeaderboardModelStat // 当月按模型合并，Users 为精确去重值
	users      map[int64]struct{}
	modelUsers map[string]map[int64]struct{}
	daily      []LeaderboardBucketPoint
	prevEnd    time.Time
	prev       []LeaderboardModelStat // 上月同期（月初 → 与当月相同的已过时长）
	prevUsers  int64
}

// ModelLeaderboardService 模型调用量排行榜。
//
// 为避免每分钟全表扫描 usage_logs：已结束月份做成小时级快照，当月只查月初至今（走 created_at 索引），
// 每次请求在内存中合并；同一数据块的并发刷新经 singleflight 合并为一次查询。
type ModelLeaderboardService struct {
	repo ModelLeaderboardRepository
	now  func() time.Time
	sf   singleflight.Group

	mu      sync.Mutex
	history map[string]*lbHistory
	live    map[string]*lbLive
}

func NewModelLeaderboardService(repo ModelLeaderboardRepository) *ModelLeaderboardService {
	return &ModelLeaderboardService{
		repo:    repo,
		now:     timezone.Now,
		history: map[string]*lbHistory{},
		live:    map[string]*lbLive{},
	}
}

// ParseLeaderboardMonth 解析 YYYY-MM；空串 = 当月。返回配置时区下该月起止。
func ParseLeaderboardMonth(month string, now time.Time) (time.Time, time.Time, error) {
	loc := timezone.Location()
	now = now.In(loc)
	if month == "" {
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, loc)
		return start, start.AddDate(0, 1, 0), nil
	}
	t, err := time.ParseInLocation("2006-01", month, loc)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid month %q, want YYYY-MM", month)
	}
	return t, t.AddDate(0, 1, 0), nil
}

// Get 返回排行榜数据。month 为空取当月；metric 为排名口径（空 = requests）；includeCost 控制是否返回费用字段。
// 缓存的是与口径无关的原始聚合，切换 metric 只影响内存中的排序与组装，不产生额外查询。
func (s *ModelLeaderboardService) Get(ctx context.Context, source, month, metric string, includeCost bool) (*ModelLeaderboardResponse, error) {
	source = usagestats.NormalizeModelSource(source)
	metric, err := ParseLeaderboardMetric(metric)
	if err != nil {
		return nil, err
	}
	now := s.now()
	monthStart, monthEnd, err := ParseLeaderboardMonth(month, now)
	if err != nil {
		return nil, err
	}
	curStart, curEnd, _ := ParseLeaderboardMonth("", now)

	hist, err := s.loadHistory(ctx, source, curStart, now)
	if err != nil {
		return nil, err
	}
	live, err := s.loadLive(ctx, source, curStart, curEnd, now)
	if err != nil {
		return nil, err
	}
	var daily []LeaderboardBucketPoint
	switch {
	case monthStart.Equal(curStart):
		daily = live.daily
	case monthStart.Before(curStart):
		if daily, err = s.loadClosedDaily(ctx, source, hist, monthStart, monthEnd); err != nil {
			return nil, err
		}
	}

	resp := assembleLeaderboard(source, metric, monthStart, monthEnd, now, hist, live, daily)
	if !includeCost {
		redactLeaderboard(resp)
	}
	return resp, nil
}

func lbDetachedCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	// singleflight 的结果由多个请求共享，不能因首个请求断开而整体失败。
	return context.WithTimeout(context.WithoutCancel(ctx), leaderboardQueryLimit)
}

func (s *ModelLeaderboardService) cachedHistory(source string, boundary, now time.Time) *lbHistory {
	s.mu.Lock()
	defer s.mu.Unlock()
	h := s.history[source]
	if h != nil && h.boundary.Equal(boundary) && now.Sub(h.builtAt) < leaderboardHistoryTTL {
		return h
	}
	return nil
}

func (s *ModelLeaderboardService) loadHistory(ctx context.Context, source string, boundary, now time.Time) (*lbHistory, error) {
	if h := s.cachedHistory(source, boundary, now); h != nil {
		return h, nil
	}
	// key 带上边界：跨月瞬间的请求不能复用仍在为旧月份构建的结果
	v, err, _ := s.sf.Do("history|"+source+"|"+boundary.Format(time.RFC3339), func() (any, error) {
		if h := s.cachedHistory(source, boundary, now); h != nil {
			return h, nil
		}
		qctx, cancel := lbDetachedCtx(ctx)
		defer cancel()
		h, err := s.buildHistory(qctx, source, boundary, now)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		if old := s.history[source]; old == nil || !old.boundary.After(h.boundary) {
			s.history[source] = h
		}
		s.mu.Unlock()
		return h, nil
	})
	if err != nil {
		return nil, err
	}
	h, ok := v.(*lbHistory)
	if !ok {
		return nil, fmt.Errorf("model leaderboard: unexpected history type %T", v)
	}
	return h, nil
}

func (s *ModelLeaderboardService) buildHistory(ctx context.Context, source string, boundary, now time.Time) (*lbHistory, error) {
	farPast := time.Date(2000, 1, 1, 0, 0, 0, 0, timezone.Location())
	stats, err := s.repo.GetMonthModelStats(ctx, farPast, boundary, source)
	if err != nil {
		return nil, fmt.Errorf("history month stats: %w", err)
	}
	monthUsers, err := s.repo.GetMonthActiveUsers(ctx, farPast, boundary)
	if err != nil {
		return nil, fmt.Errorf("history month users: %w", err)
	}
	pairs, err := s.repo.GetModelUserPairs(ctx, farPast, boundary, source)
	if err != nil {
		return nil, fmt.Errorf("history model users: %w", err)
	}

	h := &lbHistory{
		boundary:   boundary,
		builtAt:    now,
		byMonth:    map[string][]LeaderboardModelStat{},
		monthUsers: monthUsers,
		daily:      map[string][]LeaderboardBucketPoint{},
	}
	for _, st := range stats {
		h.byMonth[st.Month] = append(h.byMonth[st.Month], st)
	}
	for m := range h.byMonth {
		h.months = append(h.months, m)
	}
	sort.Strings(h.months)
	h.modelUsers, h.allUsers = lbUserSets(pairs)
	h.merged = lbMergeStats(stats)
	for i := range h.merged {
		h.merged[i].Users = int64(len(h.modelUsers[h.merged[i].Model]))
	}
	return h, nil
}

func (s *ModelLeaderboardService) cachedLive(source string, monthStart, now time.Time) *lbLive {
	s.mu.Lock()
	defer s.mu.Unlock()
	l := s.live[source]
	if l != nil && l.monthStart.Equal(monthStart) && now.Sub(l.builtAt) < leaderboardLiveTTL {
		return l
	}
	return nil
}

func (s *ModelLeaderboardService) loadLive(ctx context.Context, source string, monthStart, monthEnd, now time.Time) (*lbLive, error) {
	if l := s.cachedLive(source, monthStart, now); l != nil {
		return l, nil
	}
	v, err, _ := s.sf.Do("live|"+source+"|"+monthStart.Format(time.RFC3339), func() (any, error) {
		if l := s.cachedLive(source, monthStart, now); l != nil {
			return l, nil
		}
		qctx, cancel := lbDetachedCtx(ctx)
		defer cancel()
		l, err := s.buildLive(qctx, source, monthStart, monthEnd, now)
		if err != nil {
			return nil, err
		}
		s.mu.Lock()
		if old := s.live[source]; old == nil || !old.monthStart.After(l.monthStart) {
			s.live[source] = l
		}
		s.mu.Unlock()
		return l, nil
	})
	if err != nil {
		return nil, err
	}
	l, ok := v.(*lbLive)
	if !ok {
		return nil, fmt.Errorf("model leaderboard: unexpected live type %T", v)
	}
	return l, nil
}

func (s *ModelLeaderboardService) buildLive(ctx context.Context, source string, monthStart, monthEnd, now time.Time) (*lbLive, error) {
	stats, err := s.repo.GetMonthModelStats(ctx, monthStart, monthEnd, source)
	if err != nil {
		return nil, fmt.Errorf("current month stats: %w", err)
	}
	pairs, err := s.repo.GetModelUserPairs(ctx, monthStart, monthEnd, source)
	if err != nil {
		return nil, fmt.Errorf("current month model users: %w", err)
	}
	daily, err := s.repo.GetDailyModelUsage(ctx, monthStart, monthEnd, source)
	if err != nil {
		return nil, fmt.Errorf("current month daily: %w", err)
	}

	// 对比期 = 上月同期：上月月初起，经过与当月相同的时长（不超过上月末）。
	prevStart := monthStart.AddDate(0, -1, 0)
	prevEnd := prevStart.Add(now.Sub(monthStart))
	if prevEnd.After(monthStart) {
		prevEnd = monthStart
	}
	prevStats, err := s.repo.GetMonthModelStats(ctx, prevStart, prevEnd, source)
	if err != nil {
		return nil, fmt.Errorf("prev period stats: %w", err)
	}
	prevUsers, err := s.repo.GetMonthActiveUsers(ctx, prevStart, prevEnd)
	if err != nil {
		return nil, fmt.Errorf("prev period users: %w", err)
	}

	l := &lbLive{
		builtAt:    now,
		monthStart: monthStart,
		month:      monthStart.Format("2006-01"),
		daily:      daily,
		prevEnd:    prevEnd,
		prev:       lbMergeStats(prevStats),
	}
	for _, n := range prevUsers {
		l.prevUsers += n
	}
	l.modelUsers, l.users = lbUserSets(pairs)
	l.merged = lbMergeStats(stats)
	for i := range l.merged {
		l.merged[i].Month = l.month
		l.merged[i].Users = int64(len(l.modelUsers[l.merged[i].Model]))
	}
	return l, nil
}

func (s *ModelLeaderboardService) loadClosedDaily(ctx context.Context, source string, h *lbHistory, start, end time.Time) ([]LeaderboardBucketPoint, error) {
	key := start.Format("2006-01")
	h.dailyMu.Lock()
	pts, ok := h.daily[key]
	h.dailyMu.Unlock()
	if ok {
		return pts, nil
	}
	v, err, _ := s.sf.Do("daily|"+source+"|"+h.boundary.Format(time.RFC3339)+"|"+key, func() (any, error) {
		h.dailyMu.Lock()
		cached, hit := h.daily[key]
		h.dailyMu.Unlock()
		if hit {
			return cached, nil
		}
		qctx, cancel := lbDetachedCtx(ctx)
		defer cancel()
		pts, err := s.repo.GetDailyModelUsage(qctx, start, end, source)
		if err != nil {
			return nil, fmt.Errorf("daily trend %s: %w", key, err)
		}
		h.dailyMu.Lock()
		h.daily[key] = pts
		h.dailyMu.Unlock()
		return pts, nil
	})
	if err != nil {
		return nil, err
	}
	if pts, ok = v.([]LeaderboardBucketPoint); !ok {
		return nil, fmt.Errorf("model leaderboard: unexpected daily type %T", v)
	}
	return pts, nil
}

// ---------- 组装 ----------

func assembleLeaderboard(source, metric string, monthStart, monthEnd, now time.Time, h *lbHistory, l *lbLive, daily []LeaderboardBucketPoint) *ModelLeaderboardResponse {
	loc := timezone.Location()
	monthKey := monthStart.Format("2006-01")
	isCurrent := monthStart.Equal(l.monthStart)

	monthData := func(key string) ([]LeaderboardModelStat, int64) {
		if key == l.month {
			return l.merged, int64(len(l.users))
		}
		return h.byMonth[key], h.monthUsers[key]
	}

	curRows, curUsers := monthData(monthKey)
	var prevRows []LeaderboardModelStat
	var prevUsers int64
	periodEnd, labelEnd := monthEnd, monthEnd
	if isCurrent {
		prevRows, prevUsers = l.prev, l.prevUsers
		periodEnd, labelEnd = now, now
	} else {
		prevRows, prevUsers = monthData(monthStart.AddDate(0, -1, 0).Format("2006-01"))
	}
	if prevRows == nil {
		prevRows = []LeaderboardModelStat{} // 非 nil：对比期无数据的模型标记为新上榜
	}

	resp := &ModelLeaderboardResponse{
		Source:      source,
		Metric:      metric,
		Timezone:    timezone.Name(),
		GeneratedAt: now.In(loc).Format(time.RFC3339),
		Month:       monthKey,
		IsCurrent:   isCurrent,
		CostVisible: true,
	}

	prevSummary := lbSummarize(prevRows, prevUsers, loc)
	resp.Monthly = LeaderboardPeriod{
		Label:   monthKey,
		Start:   monthStart.Format(time.RFC3339),
		End:     periodEnd.In(loc).Format(time.RFC3339),
		Summary: lbSummarize(curRows, curUsers, loc),
		Prev:    &prevSummary,
		Ranking: buildLeaderboardRanking(curRows, prevRows, metric, loc),
		Trend:   lbBuildDailySeries(daily, monthStart, labelEnd, metric, loc),
	}

	// 历史累计 = 已结束月份快照 ⊕ 当月实时；去重用户用集合并集精确合并。
	allRows := lbMergeStats(h.merged, l.merged)
	for i := range allRows {
		m := allRows[i].Model
		allRows[i].Users = int64(len(h.modelUsers[m]) + lbCountMissing(l.modelUsers[m], h.modelUsers[m]))
	}
	allUsers := int64(len(h.allUsers) + lbCountMissing(l.users, h.allUsers))

	months := make([]string, 0, len(h.months)+1)
	months = append(months, h.months...)
	if len(l.merged) > 0 && (len(months) == 0 || months[len(months)-1] != l.month) {
		months = append(months, l.month)
	}
	monthly := make([]LeaderboardBucketPoint, 0, len(months)*8)
	for _, m := range months {
		rows, _ := monthData(m)
		for _, r := range rows {
			monthly = append(monthly, LeaderboardBucketPoint{Bucket: m, Model: r.Model, Requests: r.Requests, TotalTokens: r.TotalTokens})
		}
	}

	var beforeMonth []LeaderboardModelStat // 当月视图：对比截至上月末的累计，标出本月带来的排名变化
	if isCurrent {
		beforeMonth = append([]LeaderboardModelStat{}, h.merged...)
	}
	allSummary := lbSummarize(allRows, allUsers, loc)
	resp.AllTime = LeaderboardPeriod{
		Label:   "all",
		Start:   allSummary.FirstAt,
		End:     now.In(loc).Format(time.RFC3339),
		Summary: allSummary,
		Ranking: buildLeaderboardRanking(allRows, beforeMonth, metric, loc),
		Trend:   lbBuildBucketSeries(monthly, months, metric),
	}

	resp.MonthRows = make([]LeaderboardMonthRow, 0, len(months))
	resp.Months = make([]string, 0, len(months)+1)
	for i := len(months) - 1; i >= 0; i-- { // 新月份在前
		rows, users := monthData(months[i])
		resp.MonthRows = append(resp.MonthRows, lbMonthRow(months[i], rows, users, metric))
		resp.Months = append(resp.Months, months[i])
	}
	if len(resp.Months) == 0 || resp.Months[0] != l.month {
		resp.Months = append([]string{l.month}, resp.Months...)
	}
	return resp
}

func lbUserSets(pairs []LeaderboardModelUser) (map[string]map[int64]struct{}, map[int64]struct{}) {
	byModel := map[string]map[int64]struct{}{}
	all := map[int64]struct{}{}
	for _, p := range pairs {
		set := byModel[p.Model]
		if set == nil {
			set = map[int64]struct{}{}
			byModel[p.Model] = set
		}
		set[p.UserID] = struct{}{}
		all[p.UserID] = struct{}{}
	}
	return byModel, all
}

// lbCountMissing 返回 a 中不在 b 里的元素个数，即 |a ∪ b| - |b|。
func lbCountMissing(a, b map[int64]struct{}) int {
	n := 0
	for id := range a {
		if _, ok := b[id]; !ok {
			n++
		}
	}
	return n
}

// lbMergeStats 按模型合并多组统计（返回新切片，不修改入参）。Users 为简单相加，需要精确值时由调用方覆盖。
func lbMergeStats(groups ...[]LeaderboardModelStat) []LeaderboardModelStat {
	idx := map[string]int{}
	out := make([]LeaderboardModelStat, 0, 32)
	for _, g := range groups {
		for _, r := range g {
			i, ok := idx[r.Model]
			if !ok {
				r.Month = ""
				idx[r.Model] = len(out)
				out = append(out, r)
				continue
			}
			m := &out[i]
			m.Requests += r.Requests
			m.InputTokens += r.InputTokens
			m.OutputTokens += r.OutputTokens
			m.CacheCreationTokens += r.CacheCreationTokens
			m.CacheReadTokens += r.CacheReadTokens
			m.TotalTokens += r.TotalTokens
			m.Users += r.Users
			m.Images += r.Images
			m.DurationMsSum += r.DurationMsSum
			m.DurationCount += r.DurationCount
			m.FirstTokenMsSum += r.FirstTokenMsSum
			m.FirstTokenCount += r.FirstTokenCount
			m.Cost += r.Cost
			m.ActualCost += r.ActualCost
			if r.FirstUsedAt.Before(m.FirstUsedAt) {
				m.FirstUsedAt = r.FirstUsedAt
			}
			if r.LastUsedAt.After(m.LastUsedAt) {
				m.LastUsedAt = r.LastUsedAt
			}
		}
	}
	return out
}

func lbSummarize(rows []LeaderboardModelStat, users int64, loc *time.Location) LeaderboardSummary {
	var s LeaderboardSummary
	var cost, actual float64
	var first, last time.Time
	for _, r := range rows {
		s.Requests += r.Requests
		s.TotalTokens += r.TotalTokens
		s.InputTokens += r.InputTokens
		s.OutputTokens += r.OutputTokens
		s.CacheReadTokens += r.CacheReadTokens
		cost += r.Cost
		actual += r.ActualCost
		s.Models++
		if first.IsZero() || (!r.FirstUsedAt.IsZero() && r.FirstUsedAt.Before(first)) {
			first = r.FirstUsedAt
		}
		if r.LastUsedAt.After(last) {
			last = r.LastUsedAt
		}
	}
	s.Users = users
	if !first.IsZero() {
		s.FirstAt = first.In(loc).Format(time.RFC3339)
	}
	if !last.IsZero() {
		s.LastAt = last.In(loc).Format(time.RFC3339)
	}
	s.Cost, s.ActualCost = &cost, &actual
	return s
}

func lbSafeRatio(a, b int64) float64 {
	if b <= 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func lbAvg(sum, count int64) float64 {
	if count <= 0 {
		return 0
	}
	return float64(sum) / float64(count)
}

func lbFormatTime(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	return t.In(loc).Format(time.RFC3339)
}

// buildLeaderboardRanking 按 metric 排序并计算占比/环比。prevRows 为 nil 表示无对比期（不标新上榜）。
func buildLeaderboardRanking(rows, prevRows []LeaderboardModelStat, metric string, loc *time.Location) []LeaderboardRankItem {
	sorted := append([]LeaderboardModelStat(nil), rows...)
	sortLeaderboardRows(sorted, metric)
	prevSorted := append([]LeaderboardModelStat(nil), prevRows...)
	sortLeaderboardRows(prevSorted, metric)
	prevIndex := make(map[string]int, len(prevSorted))
	prevByModel := make(map[string]LeaderboardModelStat, len(prevSorted))
	for i, r := range prevSorted {
		prevIndex[r.Model] = i + 1
		prevByModel[r.Model] = r
	}

	var totalReq, totalTok int64
	for _, r := range sorted {
		totalReq += r.Requests
		totalTok += r.TotalTokens
	}
	out := make([]LeaderboardRankItem, 0, len(sorted))
	for i, r := range sorted {
		cost, actual := r.Cost, r.ActualCost
		prompt := r.InputTokens + r.CacheReadTokens + r.CacheCreationTokens
		item := LeaderboardRankItem{
			Rank:                i + 1,
			Model:               r.Model,
			Requests:            r.Requests,
			RequestShare:        lbSafeRatio(r.Requests, totalReq),
			TotalTokens:         r.TotalTokens,
			TokenShare:          lbSafeRatio(r.TotalTokens, totalTok),
			InputTokens:         r.InputTokens,
			OutputTokens:        r.OutputTokens,
			CacheCreationTokens: r.CacheCreationTokens,
			CacheReadTokens:     r.CacheReadTokens,
			CacheHitRate:        lbSafeRatio(r.CacheReadTokens, prompt),
			Users:               r.Users,
			Images:              r.Images,
			AvgDurationMs:       lbAvg(r.DurationMsSum, r.DurationCount),
			AvgFirstTokenMs:     lbAvg(r.FirstTokenMsSum, r.FirstTokenCount),
			FirstUsedAt:         lbFormatTime(r.FirstUsedAt, loc),
			LastUsedAt:          lbFormatTime(r.LastUsedAt, loc),
			Cost:                &cost,
			ActualCost:          &actual,
		}
		if prevRows != nil {
			if pr, ok := prevIndex[r.Model]; ok {
				rank := pr
				prev := prevByModel[r.Model]
				req, tok := prev.Requests, prev.TotalTokens
				item.PrevRank = &rank
				item.PrevRequests = &req
				item.PrevTokens = &tok
				item.RequestsGrowth = lbGrowth(r.Requests, req)
				item.TokensGrowth = lbGrowth(r.TotalTokens, tok)
			}
		}
		out = append(out, item)
	}
	return out
}

func lbGrowth(cur, prev int64) *float64 {
	if prev <= 0 {
		return nil
	}
	g := float64(cur-prev) / float64(prev)
	return &g
}

// lbMetricKeys 返回 (主排序键, 次排序键)：requests 口径次数相同比 Token，tokens 口径反之。
func lbMetricKeys(r LeaderboardModelStat, metric string) (int64, int64) {
	if metric == LeaderboardMetricTokens {
		return r.TotalTokens, r.Requests
	}
	return r.Requests, r.TotalTokens
}

func sortLeaderboardRows(rows []LeaderboardModelStat, metric string) {
	sort.SliceStable(rows, func(i, j int) bool {
		pi, si := lbMetricKeys(rows[i], metric)
		pj, sj := lbMetricKeys(rows[j], metric)
		if pi != pj {
			return pi > pj
		}
		if si != sj {
			return si > sj
		}
		return rows[i].Model < rows[j].Model
	})
}

func lbBuildDailySeries(points []LeaderboardBucketPoint, start, end time.Time, metric string, loc *time.Location) LeaderboardSeries {
	labels := make([]string, 0, 31)
	for d := start.In(loc); d.Before(end); d = d.AddDate(0, 0, 1) {
		labels = append(labels, d.Format("2006-01-02"))
	}
	return lbBuildBucketSeries(points, labels, metric)
}

// lbBuildBucketSeries 取区间内按 metric 的 Top N 模型各自一条序列，其余并入 __other__。
func lbBuildBucketSeries(points []LeaderboardBucketPoint, labels []string, metric string) LeaderboardSeries {
	idx := make(map[string]int, len(labels))
	for i, l := range labels {
		idx[l] = i
	}
	totals := map[string]int64{}
	for _, p := range points {
		if metric == LeaderboardMetricTokens {
			totals[p.Model] += p.TotalTokens
		} else {
			totals[p.Model] += p.Requests
		}
	}
	models := make([]string, 0, len(totals))
	for m := range totals {
		models = append(models, m)
	}
	sort.Slice(models, func(i, j int) bool {
		if totals[models[i]] != totals[models[j]] {
			return totals[models[i]] > totals[models[j]]
		}
		return models[i] < models[j]
	})
	top := models
	hasOther := false
	if len(models) > leaderboardTrendTopN {
		top = models[:leaderboardTrendTopN]
		hasOther = true
	}
	pos := make(map[string]int, len(top)+1)
	datasets := make([]LeaderboardSeriesDetail, 0, len(top)+1)
	for i, m := range top {
		pos[m] = i
		datasets = append(datasets, LeaderboardSeriesDetail{Model: m, Requests: make([]int64, len(labels)), Tokens: make([]int64, len(labels))})
	}
	if hasOther {
		pos[leaderboardOther] = len(datasets)
		datasets = append(datasets, LeaderboardSeriesDetail{Model: leaderboardOther, Requests: make([]int64, len(labels)), Tokens: make([]int64, len(labels))})
	}
	for _, p := range points {
		li, ok := idx[p.Bucket]
		if !ok {
			continue
		}
		di, ok := pos[p.Model]
		if !ok {
			di = pos[leaderboardOther]
		}
		datasets[di].Requests[li] += p.Requests
		datasets[di].Tokens[li] += p.TotalTokens
	}
	return LeaderboardSeries{Labels: labels, Datasets: datasets}
}

func lbMonthRow(month string, rows []LeaderboardModelStat, users int64, metric string) LeaderboardMonthRow {
	row := LeaderboardMonthRow{Month: month, Users: users, Models: int64(len(rows))}
	var cost, actual float64
	for _, r := range rows {
		row.Requests += r.Requests
		row.TotalTokens += r.TotalTokens
		cost += r.Cost
		actual += r.ActualCost
	}
	if len(rows) > 0 {
		sorted := append([]LeaderboardModelStat(nil), rows...)
		sortLeaderboardRows(sorted, metric)
		top := sorted[0]
		row.TopModel = top.Model
		if metric == LeaderboardMetricTokens {
			row.TopModelShare = lbSafeRatio(top.TotalTokens, row.TotalTokens)
		} else {
			row.TopModelShare = lbSafeRatio(top.Requests, row.Requests)
		}
	}
	row.Cost, row.ActualCost = &cost, &actual
	return row
}

// redactLeaderboard 就地抹掉所有费用字段（响应每次新建，不共享）。
func redactLeaderboard(resp *ModelLeaderboardResponse) {
	resp.CostVisible = false
	lbRedactPeriod(&resp.Monthly)
	lbRedactPeriod(&resp.AllTime)
	for i := range resp.MonthRows {
		resp.MonthRows[i].Cost, resp.MonthRows[i].ActualCost = nil, nil
	}
}

func lbRedactPeriod(p *LeaderboardPeriod) {
	p.Summary.Cost, p.Summary.ActualCost = nil, nil
	if p.Prev != nil {
		p.Prev.Cost, p.Prev.ActualCost = nil, nil
	}
	for i := range p.Ranking {
		p.Ranking[i].Cost, p.Ranking[i].ActualCost = nil, nil
	}
}
