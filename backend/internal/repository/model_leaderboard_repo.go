package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// modelLeaderboardRepository 模型调用量排行榜的只读聚合查询。
//
// 全部直接聚合 usage_logs：月份/日期分桶依赖连接级 TimeZone（DSNWithTimezone 注入配置时区），
// 与仪表盘趋势（TO_CHAR(created_at, ...)）同口径。
type modelLeaderboardRepository struct {
	db *sql.DB
}

// NewModelLeaderboardRepository 创建排行榜仓储。
func NewModelLeaderboardRepository(db *sql.DB) service.ModelLeaderboardRepository {
	return &modelLeaderboardRepository{db: db}
}

const leaderboardTokensExpr = "(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens)"

func (r *modelLeaderboardRepository) GetModelRanking(ctx context.Context, start, end time.Time, source string) (rows []service.LeaderboardModelRow, err error) {
	modelExpr := resolveModelDimensionExpression(source)
	query := fmt.Sprintf(`
		SELECT
			%[1]s AS model,
			COUNT(*) AS requests,
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cache_creation_tokens), 0),
			COALESCE(SUM(cache_read_tokens), 0),
			COALESCE(SUM(%[2]s), 0),
			COUNT(DISTINCT user_id),
			COALESCE(SUM(image_count), 0),
			COALESCE(AVG(duration_ms) FILTER (WHERE duration_ms > 0), 0),
			COALESCE(AVG(first_token_ms) FILTER (WHERE first_token_ms > 0), 0),
			COALESCE(SUM(total_cost), 0),
			COALESCE(SUM(actual_cost), 0),
			MIN(created_at),
			MAX(created_at)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY 1
		ORDER BY requests DESC, 7 DESC
	`, modelExpr, leaderboardTokensExpr)

	res, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := res.Close(); cerr != nil && err == nil {
			err = cerr
			rows = nil
		}
	}()
	rows = make([]service.LeaderboardModelRow, 0, 32)
	for res.Next() {
		var row service.LeaderboardModelRow
		if err = res.Scan(
			&row.Model, &row.Requests,
			&row.InputTokens, &row.OutputTokens, &row.CacheCreationTokens, &row.CacheReadTokens, &row.TotalTokens,
			&row.Users, &row.Images, &row.AvgDurationMs, &row.AvgFirstTokenMs,
			&row.Cost, &row.ActualCost, &row.FirstUsedAt, &row.LastUsedAt,
		); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, res.Err()
}

func (r *modelLeaderboardRepository) GetTotals(ctx context.Context, start, end time.Time, source string) (*service.LeaderboardTotals, error) {
	modelExpr := resolveModelDimensionExpression(source)
	query := fmt.Sprintf(`
		SELECT
			COUNT(*),
			COALESCE(SUM(%[2]s), 0),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cache_read_tokens), 0),
			COUNT(DISTINCT user_id),
			COUNT(DISTINCT %[1]s),
			COALESCE(SUM(total_cost), 0),
			COALESCE(SUM(actual_cost), 0),
			MIN(created_at),
			MAX(created_at)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`, modelExpr, leaderboardTokensExpr)
	var t service.LeaderboardTotals
	var first, last sql.NullTime
	if err := r.db.QueryRowContext(ctx, query, start, end).Scan(
		&t.Requests, &t.TotalTokens, &t.InputTokens, &t.OutputTokens, &t.CacheReadTokens,
		&t.Users, &t.Models, &t.Cost, &t.ActualCost, &first, &last,
	); err != nil {
		return nil, err
	}
	if first.Valid {
		v := first.Time
		t.FirstAt = &v
	}
	if last.Valid {
		v := last.Time
		t.LastAt = &v
	}
	return &t, nil
}

// GetBucketedModelUsage 按时间桶 × 模型聚合（bucket: month → YYYY-MM；day → YYYY-MM-DD）。
func (r *modelLeaderboardRepository) GetBucketedModelUsage(ctx context.Context, start, end time.Time, source, bucket string) (points []service.LeaderboardBucketPoint, err error) {
	format := "YYYY-MM-DD"
	if bucket == "month" {
		format = "YYYY-MM"
	}
	modelExpr := resolveModelDimensionExpression(source)
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, '%[1]s') AS bucket,
			%[2]s AS model,
			COUNT(*),
			COALESCE(SUM(%[3]s), 0),
			COUNT(DISTINCT user_id)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY 1, 2
		ORDER BY 1 ASC, 3 DESC
	`, format, modelExpr, leaderboardTokensExpr)

	res, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := res.Close(); cerr != nil && err == nil {
			err = cerr
			points = nil
		}
	}()
	points = make([]service.LeaderboardBucketPoint, 0, 64)
	for res.Next() {
		var p service.LeaderboardBucketPoint
		if err = res.Scan(&p.Bucket, &p.Model, &p.Requests, &p.TotalTokens, &p.Users); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, res.Err()
}

// GetMonthlyTotals 按月汇总（活跃用户需在月粒度去重，不能由模型行相加得到）。
func (r *modelLeaderboardRepository) GetMonthlyTotals(ctx context.Context, source string) (out []service.LeaderboardMonthTotal, err error) {
	modelExpr := resolveModelDimensionExpression(source)
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, 'YYYY-MM') AS month,
			COUNT(*),
			COALESCE(SUM(%[2]s), 0),
			COUNT(DISTINCT user_id),
			COUNT(DISTINCT %[1]s),
			COALESCE(SUM(total_cost), 0),
			COALESCE(SUM(actual_cost), 0)
		FROM usage_logs
		GROUP BY 1
		ORDER BY 1 ASC
	`, modelExpr, leaderboardTokensExpr)
	res, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := res.Close(); cerr != nil && err == nil {
			err = cerr
			out = nil
		}
	}()
	out = make([]service.LeaderboardMonthTotal, 0, 12)
	for res.Next() {
		var m service.LeaderboardMonthTotal
		if err = res.Scan(&m.Month, &m.Requests, &m.TotalTokens, &m.Users, &m.Models, &m.Cost, &m.ActualCost); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, res.Err()
}
