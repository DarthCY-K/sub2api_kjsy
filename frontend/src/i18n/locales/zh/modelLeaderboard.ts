// 模型调用量排行榜（DarthCY 定制）
export default {
  modelLeaderboard: {
    title: '模型排行榜',
    description: '全站模型用量：当月与历史累计',
    nav: '排行榜',
    loadFailed: '加载排行榜失败',
    empty: '暂无调用数据',
    refresh: '刷新',
    tabs: { monthly: '当月', allTime: '历史累计', months: '月度明细' },
    source: { label: '模型口径', requested: '请求模型', upstream: '上游模型' },
    month: '统计月份',
    vendor: { label: '供应商', all: '全部供应商', other: '其他', hint: '按模型名称关键词归类，如 deepseek-* → DeepSeek、k3 → Moonshot、opus → Claude' },
    metric: { label: '统计口径', requests: '调用次数', tokens: 'Token 数', users: '活跃用户' },
    summary: {
      requests: '调用次数',
      tokens: 'Token 总量',
      users: '活跃用户',
      models: '模型数',
      cost: '标准计费',
      actualCost: '实际扣费',
      vsPrev: '较上月同期',
      vsPrevMonth: '较上月',
      since: '自 {date} 起',
      range: '{start} ~ {end}'
    },
    charts: {
      share: '{metric}占比',
      topBar: 'Top 10 · {metric}',
      trendDaily: '每日趋势 · {metric}',
      trendMonthly: '每月趋势 · {metric}',
      tokenMix: 'Token 构成（Top 10）',
      input: '输入',
      output: '输出',
      cacheCreate: '缓存写入',
      cacheRead: '缓存读取',
      other: '其他'
    },
    table: {
      title: '完整排行',
      rank: '排名',
      model: '模型',
      requests: '调用次数',
      share: '占比',
      tokens: 'Token',
      input: '输入',
      output: '输出',
      cacheHit: '缓存命中',
      users: '用户',
      latency: '平均耗时',
      ttft: '首字',
      growth: '环比',
      lastUsed: '最近调用',
      cost: '标准计费',
      actualCost: '实际扣费',
      newEntry: '新上榜',
      search: '搜索模型'
    },
    months: {
      title: '月度汇总',
      month: '月份',
      topModel: '当月第一',
      topShare: '第一占比'
    },
    footnote: '数据来自全站调用日志，当月数据每 60 秒刷新；时区 {tz}。排名按{metric}，相同时按{tie}。'
  }
}
