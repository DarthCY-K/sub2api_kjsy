/**
 * 模型调用量排行榜（DarthCY 定制）
 * GET /api/v1/model-leaderboard?source=requested|upstream&month=YYYY-MM&metric=requests|tokens&vendor=claude|...
 */
import { apiClient } from './client'

export type LeaderboardSource = 'requested' | 'upstream'
export type LeaderboardMetric = 'requests' | 'tokens'

export interface LeaderboardRankItem {
  rank: number
  model: string
  requests: number
  request_share: number
  total_tokens: number
  token_share: number
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  cache_hit_rate: number
  users: number
  images: number
  avg_duration_ms: number
  avg_first_token_ms: number
  first_used_at: string
  last_used_at: string
  prev_rank: number | null
  prev_requests: number | null
  requests_growth: number | null
  prev_tokens: number | null
  tokens_growth: number | null
  cost?: number
  actual_cost?: number
}

export interface LeaderboardSummary {
  requests: number
  total_tokens: number
  input_tokens: number
  output_tokens: number
  cache_read_tokens: number
  users: number
  models: number
  first_at?: string
  last_at?: string
  cost?: number
  actual_cost?: number
}

export interface LeaderboardSeries {
  labels: string[]
  datasets: Array<{ model: string; requests: number[]; tokens: number[] }>
}

export interface LeaderboardUserModel {
  model: string
  requests: number
  total_tokens: number
  /** 该模型占此用户当前口径的比例 */
  share: number
  cost?: number
  actual_cost?: number
}

/** 用户用量排行；非管理员看到的他人 name 为脱敏名，且无 user_id / email */
export interface LeaderboardUserItem {
  rank: number
  user_id?: number
  name: string
  email?: string
  deleted: boolean
  is_self: boolean
  requests: number
  request_share: number
  total_tokens: number
  token_share: number
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  models: LeaderboardUserModel[]
  last_used_at: string
  cost?: number
  actual_cost?: number
}

export interface LeaderboardPeriod {
  label: string
  start: string
  end: string
  summary: LeaderboardSummary
  prev?: LeaderboardSummary
  ranking: LeaderboardRankItem[]
  trend: LeaderboardSeries
  /** 前 100 名 */
  user_ranking: LeaderboardUserItem[]
  /** 本人不在前 100 名时单独返回 */
  user_self?: LeaderboardUserItem
}

export interface LeaderboardMonthRow {
  month: string
  requests: number
  total_tokens: number
  users: number
  models: number
  top_model: string
  top_model_share: number
  cost?: number
  actual_cost?: number
}

export interface ModelLeaderboardResponse {
  source: LeaderboardSource
  metric: LeaderboardMetric
  /** 当前筛选的供应商，空串 = 全部 */
  vendor: string
  /** 有数据的供应商（按历史累计口径降序，other 恒在最后） */
  vendors: string[]
  timezone: string
  generated_at: string
  month: string
  is_current: boolean
  months: string[]
  cost_visible: boolean
  monthly: LeaderboardPeriod
  all_time: LeaderboardPeriod
  month_rows: LeaderboardMonthRow[]
}

export const LEADERBOARD_OTHER = '__other__'

export async function getModelLeaderboard(
  params: { source?: LeaderboardSource; month?: string; metric?: LeaderboardMetric; vendor?: string } = {},
  options?: { signal?: AbortSignal }
): Promise<ModelLeaderboardResponse> {
  const { data } = await apiClient.get<ModelLeaderboardResponse>('/model-leaderboard', {
    params,
    signal: options?.signal
  })
  return data
}

export default { getModelLeaderboard }
