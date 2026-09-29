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
    rankBy: 'Rank by',
    metric: { requests: 'Requests', tokens: 'Tokens', users: 'Users' },
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
      share: 'Request share',
      topBar: 'Top 10 by requests',
      trendDaily: 'Daily requests',
      trendMonthly: 'Monthly requests',
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
      images: 'Images',
      latency: 'Avg latency',
      ttft: 'TTFT',
      growth: 'Change',
      lastUsed: 'Last used',
      firstUsed: 'First used',
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
    footnote: 'Aggregated from site-wide usage logs, cached for 60s; timezone {tz}. Ranked by requests, ties broken by tokens.'
  }
}
