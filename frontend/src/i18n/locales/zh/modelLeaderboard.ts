// 模型调用量排行榜（DarthCY 定制）
export default {
  modelLeaderboard: {
    title: '模型排行榜',
    description: '全站模型调用量：当月与历史累计',
    nav: '排行榜',
    loadFailed: '加载排行榜失败',
    empty: '暂无调用数据',
    refresh: '刷新',
    tabs: { monthly: '当月', allTime: '历史累计', months: '月度明细' },
    source: { label: '模型口径', requested: '请求模型', upstream: '上游模型' },
    month: '统计月份',
    rankBy: '排序',
    metric: { requests: '调用次数', tokens: 'Token 用量', users: '用户数' },
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
      share: '调用占比',
      topBar: 'Top 10 调用量',
      trendDaily: '每日调用趋势',
      trendMonthly: '每月调用趋势',
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
      images: '图片',
      latency: '平均耗时',
      ttft: '首字',
      growth: '环比',
      lastUsed: '最近调用',
      firstUsed: '首次调用',
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
    footnote: '数据来自全站调用日志，每 60 秒刷新缓存；时区 {tz}。排名按调用次数，次数相同按 Token。'
  }
}
