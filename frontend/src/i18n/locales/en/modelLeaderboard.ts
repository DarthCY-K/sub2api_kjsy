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
    hero: {
      currentChampion: 'Champion this month',
      monthChampion: '{month} champion',
      allTimeChampion: 'All-time champion',
      share: '{pct} of all requests',
      lead: 'Ahead of #2 by {n}'
    },
    podium: { first: 'Champion', second: 'Runner-up', third: 'Third place' },
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
      other: 'Others',
      totalRequests: 'Total'
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
      topShare: 'Top share',
      peak: 'Peak'
    },
    footnote: 'Aggregated from site-wide usage logs; current month refreshes every 60s; timezone {tz}. Ranked by requests, ties broken by tokens.'
  }
}
