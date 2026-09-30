/**
 * 模型调用量排行榜（DarthCY 定制）
 * GET /api/v1/model-leaderboard?source=requested|upstream&month=YYYY-MM
 */
import { apiClient } from './client'

export type LeaderboardSource = 'requested' | 'upstream'

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

export interface LeaderboardPeriod {
  label: string
  start: string
  end: string
  summary: LeaderboardSummary
  prev?: LeaderboardSummary
  ranking: LeaderboardRankItem[]
  trend: LeaderboardSeries
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
  params: { source?: LeaderboardSource; month?: string } = {},
  options?: { signal?: AbortSignal }
): Promise<ModelLeaderboardResponse> {
  const { data } = await apiClient.get<ModelLeaderboardResponse>('/model-leaderboard', {
    params,
    signal: options?.signal
  })
  return data
}

export default { getModelLeaderboard }
