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
// 全部直接聚合 usage_logs，且均带 created_at 区间条件（走 idx_usage_logs_created_at）。
// 月份/日期分桶依赖连接级 TimeZone（DSNWithTimezone 注入配置时区），
// 与仪表盘趋势（TO_CHAR(created_at, ...)）同口径。
type modelLeaderboardRepository struct {
	db *sql.DB
}

// NewModelLeaderboardRepository 创建排行榜仓储。
func NewModelLeaderboardRepository(db *sql.DB) service.ModelLeaderboardRepository {
	return &modelLeaderboardRepository{db: db}
}

const leaderboardTokensExpr = "(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens)"

func (r *modelLeaderboardRepository) GetMonthModelStats(ctx context.Context, start, end time.Time, source string) (out []service.LeaderboardModelStat, err error) {
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, 'YYYY-MM') AS month,
			%[1]s AS model,
			COUNT(*),
			COALESCE(SUM(input_tokens), 0),
			COALESCE(SUM(output_tokens), 0),
			COALESCE(SUM(cache_creation_tokens), 0),
			COALESCE(SUM(cache_read_tokens), 0),
			COALESCE(SUM(%[2]s), 0),
			COUNT(DISTINCT user_id),
			COALESCE(SUM(image_count), 0),
			COALESCE(SUM(duration_ms) FILTER (WHERE duration_ms > 0), 0)::BIGINT,
			COUNT(*) FILTER (WHERE duration_ms > 0),
			COALESCE(SUM(first_token_ms) FILTER (WHERE first_token_ms > 0), 0)::BIGINT,
			COUNT(*) FILTER (WHERE first_token_ms > 0),
			COALESCE(SUM(total_cost), 0),
			COALESCE(SUM(actual_cost), 0),
			MIN(created_at),
			MAX(created_at)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY 1, 2
	`, resolveModelDimensionExpression(source), leaderboardTokensExpr)

	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
			out = nil
		}
	}()
	out = make([]service.LeaderboardModelStat, 0, 64)
	for rows.Next() {
		var s service.LeaderboardModelStat
		if err = rows.Scan(
			&s.Month, &s.Model, &s.Requests,
			&s.InputTokens, &s.OutputTokens, &s.CacheCreationTokens, &s.CacheReadTokens, &s.TotalTokens,
			&s.Users, &s.Images,
			&s.DurationMsSum, &s.DurationCount, &s.FirstTokenMsSum, &s.FirstTokenCount,
			&s.Cost, &s.ActualCost, &s.FirstUsedAt, &s.LastUsedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *modelLeaderboardRepository) GetMonthActiveUsers(ctx context.Context, start, end time.Time) (out map[string]int64, err error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT TO_CHAR(created_at, 'YYYY-MM'), COUNT(DISTINCT user_id)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY 1
	`, start, end)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
			out = nil
		}
	}()
	out = make(map[string]int64, 16)
	for rows.Next() {
		var month string
		var users int64
		if err = rows.Scan(&month, &users); err != nil {
			return nil, err
		}
		out[month] = users
	}
	return out, rows.Err()
}

func (r *modelLeaderboardRepository) GetModelUserPairs(ctx context.Context, start, end time.Time, source string) (out []service.LeaderboardModelUser, err error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT %s AS model, user_id
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`, resolveModelDimensionExpression(source))
	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
			out = nil
		}
	}()
	out = make([]service.LeaderboardModelUser, 0, 256)
	for rows.Next() {
		var p service.LeaderboardModelUser
		if err = rows.Scan(&p.Model, &p.UserID); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *modelLeaderboardRepository) GetDailyModelUsage(ctx context.Context, start, end time.Time, source string) (out []service.LeaderboardBucketPoint, err error) {
	query := fmt.Sprintf(`
		SELECT
			TO_CHAR(created_at, 'YYYY-MM-DD') AS bucket,
			%[1]s AS model,
			COUNT(*),
			COALESCE(SUM(%[2]s), 0)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
		GROUP BY 1, 2
	`, resolveModelDimensionExpression(source), leaderboardTokensExpr)
	rows, err := r.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil && err == nil {
			err = cerr
			out = nil
		}
	}()
	out = make([]service.LeaderboardBucketPoint, 0, 128)
	for rows.Next() {
		var p service.LeaderboardBucketPoint
		if err = rows.Scan(&p.Bucket, &p.Model, &p.Requests, &p.TotalTokens); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
