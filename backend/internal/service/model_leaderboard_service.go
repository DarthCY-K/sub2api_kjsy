package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// ModelLeaderboardRepository 模型调用量排行榜聚合查询（只读）。
type ModelLeaderboardRepository interface {
	GetModelRanking(ctx context.Context, start, end time.Time, source string) ([]LeaderboardModelRow, error)
	GetTotals(ctx context.Context, start, end time.Time, source string) (*LeaderboardTotals, error)
	GetBucketedModelUsage(ctx context.Context, start, end time.Time, source, bucket string) ([]LeaderboardBucketPoint, error)
	GetMonthlyTotals(ctx context.Context, source string) ([]LeaderboardMonthTotal, error)
}

// LeaderboardModelRow 单模型在区间内的聚合。
type LeaderboardModelRow struct {
	Model               string
	Requests            int64
	InputTokens         int64
	OutputTokens        int64
	CacheCreationTokens int64
	CacheReadTokens     int64
	TotalTokens         int64
	Users               int64
	Images              int64
	AvgDurationMs       float64
	AvgFirstTokenMs     float64
	Cost                float64
	ActualCost          float64
	FirstUsedAt         time.Time
	LastUsedAt          time.Time
}

// LeaderboardTotals 区间总计。
type LeaderboardTotals struct {
	Requests        int64
	TotalTokens     int64
	InputTokens     int64
	OutputTokens    int64
	CacheReadTokens int64
	Users           int64
	Models          int64
	Cost            float64
	ActualCost      float64
	FirstAt         *time.Time
	LastAt          *time.Time
}

// LeaderboardBucketPoint 时间桶 × 模型。
type LeaderboardBucketPoint struct {
	Bucket      string
	Model       string
	Requests    int64
	TotalTokens int64
	Users       int64
}

// LeaderboardMonthTotal 月度汇总。
type LeaderboardMonthTotal struct {
	Month       string
	Requests    int64
	TotalTokens int64
	Users       int64
	Models      int64
	Cost        float64
	ActualCost  float64
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
	PrevRank            *int     `json:"prev_rank"`             // 对比期排名；nil = 新上榜
	PrevRequests        *int64   `json:"prev_requests"`         // 对比期调用量
	RequestsGrowth      *float64 `json:"requests_growth"`       // 环比（小数，0.25=+25%）；对比期无数据为 nil
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
	TopModel      string   `json:"top_model"`
	TopModelShare float64  `json:"top_model_share"`
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
	leaderboardCacheTTL  = 60 * time.Second
	leaderboardTrendTopN = 8
	leaderboardOther     = "__other__"
)

type leaderboardCacheEntry struct {
	at   time.Time
	resp *ModelLeaderboardResponse
}

// ModelLeaderboardService 模型调用量排行榜。结果按 (source, month) 缓存 60s，
// 管理员/普通用户共用同一份聚合，出参时再裁剪费用字段。
type ModelLeaderboardService struct {
	repo  ModelLeaderboardRepository
	mu    sync.Mutex
	cache map[string]leaderboardCacheEntry
	now   func() time.Time
}

func NewModelLeaderboardService(repo ModelLeaderboardRepository) *ModelLeaderboardService {
	return &ModelLeaderboardService{repo: repo, cache: map[string]leaderboardCacheEntry{}, now: timezone.Now}
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

// Get 返回排行榜数据。month 为空取当月；includeCost 控制是否返回费用字段。
func (s *ModelLeaderboardService) Get(ctx context.Context, source, month string, includeCost bool) (*ModelLeaderboardResponse, error) {
	source = usagestats.NormalizeModelSource(source)
	now := s.now()
	monthStart, monthEnd, err := ParseLeaderboardMonth(month, now)
	if err != nil {
		return nil, err
	}
	key := source + "|" + monthStart.Format("2006-01")

	s.mu.Lock()
	entry, ok := s.cache[key]
	s.mu.Unlock()
	if ok && now.Sub(entry.at) < leaderboardCacheTTL {
		return redactLeaderboard(entry.resp, includeCost), nil
	}

	resp, err := s.build(ctx, source, monthStart, monthEnd, now)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[key] = leaderboardCacheEntry{at: now, resp: resp}
	for k, e := range s.cache { // 顺手清理过期项，避免月份参数把 map 撑大
		if now.Sub(e.at) >= leaderboardCacheTTL*5 {
			delete(s.cache, k)
		}
	}
	s.mu.Unlock()
	return redactLeaderboard(resp, includeCost), nil
}

func (s *ModelLeaderboardService) build(ctx context.Context, source string, monthStart, monthEnd, now time.Time) (*ModelLeaderboardResponse, error) {
	loc := timezone.Location()
	isCurrentMonth := !now.Before(monthStart) && now.Before(monthEnd)

	// 当月口径：区间截至 now；对比期 = 上月同期（月初到同一日时刻），历史月份则整月对比整月。
	curEnd := monthEnd
	prevStart := monthStart.AddDate(0, -1, 0)
	prevEnd := monthStart
	if isCurrentMonth {
		curEnd = now
		elapsed := now.Sub(monthStart)
		prevEnd = prevStart.Add(elapsed)
		if prevEnd.After(monthStart) {
			prevEnd = monthStart
		}
	}
	farPast := time.Date(2000, 1, 1, 0, 0, 0, 0, loc)
	farFuture := now.Add(24 * time.Hour)

	monthRows, err := s.repo.GetModelRanking(ctx, monthStart, curEnd, source)
	if err != nil {
		return nil, fmt.Errorf("monthly ranking: %w", err)
	}
	monthTotals, err := s.repo.GetTotals(ctx, monthStart, curEnd, source)
	if err != nil {
		return nil, fmt.Errorf("monthly totals: %w", err)
	}
	prevRows, err := s.repo.GetModelRanking(ctx, prevStart, prevEnd, source)
	if err != nil {
		return nil, fmt.Errorf("prev ranking: %w", err)
	}
	prevTotals, err := s.repo.GetTotals(ctx, prevStart, prevEnd, source)
	if err != nil {
		return nil, fmt.Errorf("prev totals: %w", err)
	}
	daily, err := s.repo.GetBucketedModelUsage(ctx, monthStart, curEnd, source, "day")
	if err != nil {
		return nil, fmt.Errorf("daily trend: %w", err)
	}
	allRows, err := s.repo.GetModelRanking(ctx, farPast, farFuture, source)
	if err != nil {
		return nil, fmt.Errorf("all-time ranking: %w", err)
	}
	allTotals, err := s.repo.GetTotals(ctx, farPast, farFuture, source)
	if err != nil {
		return nil, fmt.Errorf("all-time totals: %w", err)
	}
	monthly, err := s.repo.GetBucketedModelUsage(ctx, farPast, farFuture, source, "month")
	if err != nil {
		return nil, fmt.Errorf("monthly trend: %w", err)
	}
	monthTotalRows, err := s.repo.GetMonthlyTotals(ctx, source)
	if err != nil {
		return nil, fmt.Errorf("month totals: %w", err)
	}

	// 历史累计的“对比期”= 截至上月末的累计，用来标出本月带来的排名变化。
	var beforeMonthRows []LeaderboardModelRow
	if isCurrentMonth {
		beforeMonthRows, err = s.repo.GetModelRanking(ctx, farPast, monthStart, source)
		if err != nil {
			return nil, fmt.Errorf("pre-month ranking: %w", err)
		}
	}

	resp := &ModelLeaderboardResponse{
		Source:      source,
		Timezone:    timezone.Name(),
		GeneratedAt: now.In(loc).Format(time.RFC3339),
		Month:       monthStart.Format("2006-01"),
		IsCurrent:   isCurrentMonth,
		CostVisible: true,
	}

	prevSummary := toLeaderboardSummary(prevTotals, loc)
	resp.Monthly = LeaderboardPeriod{
		Label:   monthStart.Format("2006-01"),
		Start:   monthStart.Format(time.RFC3339),
		End:     curEnd.In(loc).Format(time.RFC3339),
		Summary: toLeaderboardSummary(monthTotals, loc),
		Prev:    &prevSummary,
		Ranking: buildLeaderboardRanking(monthRows, prevRows, loc),
		Trend:   lbBuildDailySeries(daily, monthStart, curEnd, loc),
	}
	resp.AllTime = LeaderboardPeriod{
		Label:   "all",
		Start:   lbFormatOptionalTime(allTotals.FirstAt, loc),
		End:     now.In(loc).Format(time.RFC3339),
		Summary: toLeaderboardSummary(allTotals, loc),
		Ranking: buildLeaderboardRanking(allRows, beforeMonthRows, loc),
		Trend:   lbBuildBucketSeries(monthly, lbMonthLabels(monthTotalRows)),
	}
	resp.MonthRows = lbBuildMonthRows(monthTotalRows, monthly)
	resp.Months = make([]string, 0, len(monthTotalRows))
	for i := len(monthTotalRows) - 1; i >= 0; i-- {
		resp.Months = append(resp.Months, monthTotalRows[i].Month)
	}
	if isCurrentMonth && (len(resp.Months) == 0 || resp.Months[0] != resp.Month) {
		resp.Months = append([]string{resp.Month}, resp.Months...)
	}
	return resp, nil
}

func lbFormatOptionalTime(t *time.Time, loc *time.Location) string {
	if t == nil {
		return ""
	}
	return t.In(loc).Format(time.RFC3339)
}

func toLeaderboardSummary(t *LeaderboardTotals, loc *time.Location) LeaderboardSummary {
	if t == nil {
		return LeaderboardSummary{}
	}
	cost, actual := t.Cost, t.ActualCost
	return LeaderboardSummary{
		Requests:        t.Requests,
		TotalTokens:     t.TotalTokens,
		InputTokens:     t.InputTokens,
		OutputTokens:    t.OutputTokens,
		CacheReadTokens: t.CacheReadTokens,
		Users:           t.Users,
		Models:          t.Models,
		FirstAt:         lbFormatOptionalTime(t.FirstAt, loc),
		LastAt:          lbFormatOptionalTime(t.LastAt, loc),
		Cost:            &cost,
		ActualCost:      &actual,
	}
}

func lbSafeRatio(a, b int64) float64 {
	if b <= 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func buildLeaderboardRanking(rows, prevRows []LeaderboardModelRow, loc *time.Location) []LeaderboardRankItem {
	sorted := append([]LeaderboardModelRow(nil), rows...)
	sortLeaderboardRows(sorted)
	prevSorted := append([]LeaderboardModelRow(nil), prevRows...)
	sortLeaderboardRows(prevSorted)
	prevIndex := make(map[string]int, len(prevSorted))
	prevReq := make(map[string]int64, len(prevSorted))
	for i, r := range prevSorted {
		prevIndex[r.Model] = i + 1
		prevReq[r.Model] = r.Requests
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
			AvgDurationMs:       r.AvgDurationMs,
			AvgFirstTokenMs:     r.AvgFirstTokenMs,
			FirstUsedAt:         r.FirstUsedAt.In(loc).Format(time.RFC3339),
			LastUsedAt:          r.LastUsedAt.In(loc).Format(time.RFC3339),
			Cost:                &cost,
			ActualCost:          &actual,
		}
		if prevRows != nil {
			if pr, ok := prevIndex[r.Model]; ok {
				rank := pr
				req := prevReq[r.Model]
				item.PrevRank = &rank
				item.PrevRequests = &req
				if req > 0 {
					g := float64(r.Requests-req) / float64(req)
					item.RequestsGrowth = &g
				}
			}
		}
		out = append(out, item)
	}
	return out
}

func sortLeaderboardRows(rows []LeaderboardModelRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Requests != rows[j].Requests {
			return rows[i].Requests > rows[j].Requests
		}
		if rows[i].TotalTokens != rows[j].TotalTokens {
			return rows[i].TotalTokens > rows[j].TotalTokens
		}
		return rows[i].Model < rows[j].Model
	})
}

func lbBuildDailySeries(points []LeaderboardBucketPoint, start, end time.Time, loc *time.Location) LeaderboardSeries {
	labels := make([]string, 0, 31)
	for d := start.In(loc); d.Before(end); d = d.AddDate(0, 0, 1) {
		labels = append(labels, d.Format("2006-01-02"))
	}
	return lbBuildBucketSeries(points, labels)
}

func lbMonthLabels(rows []LeaderboardMonthTotal) []string {
	labels := make([]string, 0, len(rows))
	for _, r := range rows {
		labels = append(labels, r.Month)
	}
	return labels
}

// buildBucketSeries 取区间内调用量 Top N 模型各自一条序列，其余并入 __other__。
func lbBuildBucketSeries(points []LeaderboardBucketPoint, labels []string) LeaderboardSeries {
	idx := make(map[string]int, len(labels))
	for i, l := range labels {
		idx[l] = i
	}
	totals := map[string]int64{}
	for _, p := range points {
		totals[p.Model] += p.Requests
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

func lbBuildMonthRows(totals []LeaderboardMonthTotal, points []LeaderboardBucketPoint) []LeaderboardMonthRow {
	type top struct {
		model string
		req   int64
	}
	best := map[string]top{}
	for _, p := range points {
		b := best[p.Bucket]
		if p.Requests > b.req || (p.Requests == b.req && (b.model == "" || p.Model < b.model)) {
			best[p.Bucket] = top{model: p.Model, req: p.Requests}
		}
	}
	out := make([]LeaderboardMonthRow, 0, len(totals))
	for i := len(totals) - 1; i >= 0; i-- { // 新月份在前
		t := totals[i]
		cost, actual := t.Cost, t.ActualCost
		b := best[t.Month]
		out = append(out, LeaderboardMonthRow{
			Month:         t.Month,
			Requests:      t.Requests,
			TotalTokens:   t.TotalTokens,
			Users:         t.Users,
			Models:        t.Models,
			TopModel:      b.model,
			TopModelShare: lbSafeRatio(b.req, t.Requests),
			Cost:          &cost,
			ActualCost:    &actual,
		})
	}
	return out
}

// redactLeaderboard 返回一份浅拷贝；includeCost=false 时抹掉所有费用字段。
// 缓存里的原始对象不被修改。
func redactLeaderboard(src *ModelLeaderboardResponse, includeCost bool) *ModelLeaderboardResponse {
	out := *src
	out.CostVisible = includeCost
	if includeCost {
		return &out
	}
	out.Monthly = lbRedactPeriod(src.Monthly)
	out.AllTime = lbRedactPeriod(src.AllTime)
	out.MonthRows = make([]LeaderboardMonthRow, len(src.MonthRows))
	for i, r := range src.MonthRows {
		r.Cost, r.ActualCost = nil, nil
		out.MonthRows[i] = r
	}
	return &out
}

func lbRedactPeriod(p LeaderboardPeriod) LeaderboardPeriod {
	p.Summary.Cost, p.Summary.ActualCost = nil, nil
	if p.Prev != nil {
		prev := *p.Prev
		prev.Cost, prev.ActualCost = nil, nil
		p.Prev = &prev
	}
	ranking := make([]LeaderboardRankItem, len(p.Ranking))
	for i, r := range p.Ranking {
		r.Cost, r.ActualCost = nil, nil
		ranking[i] = r
	}
	p.Ranking = ranking
	return p
}
