// Model usage leaderboard (DarthCY custom)
export default {
  modelLeaderboard: {
    title: 'Model Leaderboard',
    description: 'Site-wide model usage: this month and all time',
    nav: 'Leaderboard',
    loadFailed: 'Failed to load leaderboard',
    empty: 'No usage data yet',
    refresh: 'Refresh',
    tabs: { monthly: 'This month', allTime: 'All time', months: 'By month' },
    source: { label: 'Model', requested: 'Requested', upstream: 'Upstream' },
    month: 'Month',
    vendor: { label: 'Vendor', all: 'All vendors', other: 'Other', hint: 'Grouped by model-name keywords, e.g. deepseek-* → DeepSeek, k3 → Moonshot, opus → Claude' },
    metric: { label: 'Rank by', requests: 'Requests', tokens: 'Tokens', users: 'Active users' },
    summary: {
      requests: 'Requests',
      tokens: 'Total tokens',
      users: 'Active users',
      models: 'Models',
      cost: 'Standard cost',
      actualCost: 'Actual charged',
      vsPrev: 'vs same period last month',
      vsPrevMonth: 'vs last month',
      since: 'Since {date}',
      range: '{start} ~ {end}'
    },
    charts: {
      share: '{metric} share',
      topBar: 'Top 10 · {metric}',
      trendDaily: 'Daily trend · {metric}',
      trendMonthly: 'Monthly trend · {metric}',
      tokenMix: 'Token mix (Top 10)',
      input: 'Input',
      output: 'Output',
      cacheCreate: 'Cache write',
      cacheRead: 'Cache read',
      other: 'Others'
    },
    table: {
      title: 'Full ranking',
      rank: 'Rank',
      model: 'Model',
      requests: 'Requests',
      share: 'Share',
      tokens: 'Tokens',
      input: 'Input',
      output: 'Output',
      cacheHit: 'Cache hit',
      users: 'Users',
      latency: 'Avg latency',
      ttft: 'TTFT',
      growth: 'Change',
      lastUsed: 'Last used',
      cost: 'Standard cost',
      actualCost: 'Actual charged',
      newEntry: 'New',
      search: 'Search model'
    },
    months: {
      title: 'Monthly summary',
      month: 'Month',
      topModel: 'Top model',
      topShare: 'Top share'
    },
    footnote: 'Aggregated from site-wide usage logs; current month refreshes every 60s; timezone {tz}. Ranked by {metric}, ties broken by {tie}.'
  }
}
