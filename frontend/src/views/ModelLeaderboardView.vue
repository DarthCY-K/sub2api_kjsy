<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- Hero：标题 / 控件 / 当前冠军 -->
      <section class="relative overflow-hidden rounded-2xl bg-gradient-to-br from-primary-600 via-teal-600 to-cyan-700 p-5 text-white shadow-xl shadow-primary-500/20 md:p-6">
        <div class="pointer-events-none absolute -left-16 -top-24 h-64 w-64 rounded-full bg-white/10 blur-3xl" />
        <div class="pointer-events-none absolute -bottom-24 right-10 h-72 w-72 rounded-full bg-cyan-300/20 blur-3xl" />
        <div class="pointer-events-none absolute right-1/3 top-0 h-40 w-40 rounded-full bg-amber-300/10 blur-2xl" />

        <div class="relative flex flex-col gap-6 lg:flex-row lg:items-stretch lg:justify-between">
          <div class="flex min-w-0 flex-col gap-4">
            <div class="flex items-center gap-3">
              <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white/15 ring-1 ring-white/25 backdrop-blur">
                <Icon name="trophy" size="lg" />
              </div>
              <div class="min-w-0">
                <h1 class="text-xl font-bold tracking-tight md:text-2xl">{{ t('modelLeaderboard.title') }}</h1>
                <p class="truncate text-sm text-white/75">{{ t('modelLeaderboard.description') }}</p>
              </div>
            </div>

            <div class="flex flex-wrap items-center gap-2.5">
              <div class="inline-flex rounded-xl bg-black/15 p-1 ring-1 ring-white/15 backdrop-blur" role="tablist">
                <button
                  v-for="tab in tabs"
                  :key="tab.value"
                  type="button"
                  role="tab"
                  :aria-selected="activeTab === tab.value"
                  class="rounded-lg px-3 py-1.5 text-sm font-medium transition-all"
                  :class="activeTab === tab.value ? heroActiveClass : heroIdleClass"
                  @click="activeTab = tab.value"
                >
                  {{ tab.label }}
                </button>
              </div>

              <div class="inline-flex items-center gap-1 rounded-xl bg-black/15 p-1 ring-1 ring-white/15 backdrop-blur">
                <span class="px-1.5 text-xs text-white/70">{{ t('modelLeaderboard.source.label') }}</span>
                <button
                  v-for="s in sources"
                  :key="s"
                  type="button"
                  class="rounded-lg px-2.5 py-1 text-xs font-medium transition-all"
                  :class="source === s ? heroActiveClass : heroIdleClass"
                  @click="setSource(s)"
                >
                  {{ t(`modelLeaderboard.source.${s}`) }}
                </button>
              </div>

              <div v-if="activeTab === 'monthly' && monthOptions.length" class="inline-flex items-center gap-1.5 rounded-xl bg-black/15 py-1 pl-2.5 pr-1 ring-1 ring-white/15 backdrop-blur">
                <Icon name="calendar" size="sm" class="text-white/70" />
                <label for="lb-month" class="sr-only">{{ t('modelLeaderboard.month') }}</label>
                <select
                  id="lb-month"
                  v-model="month"
                  class="h-7 cursor-pointer rounded-lg border-0 bg-white/10 py-0 pl-2 pr-7 text-sm font-medium text-white focus:ring-2 focus:ring-white/40"
                  @change="load()"
                >
                  <option v-for="m in monthOptions" :key="m" :value="m" class="text-gray-900">{{ m }}</option>
                </select>
              </div>

              <button
                type="button"
                class="inline-flex items-center rounded-xl bg-white/15 px-3 py-1.5 text-sm font-medium ring-1 ring-white/25 backdrop-blur transition-colors hover:bg-white/25 disabled:opacity-60"
                :disabled="loading"
                @click="load(true)"
              >
                <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
                <span class="ml-1.5">{{ t('modelLeaderboard.refresh') }}</span>
              </button>
            </div>

            <p v-if="data" class="text-xs text-white/65">
              {{ periodCaption }}
            </p>
          </div>

          <!-- 冠军卡 -->
          <div
            v-if="champion"
            class="relative w-full shrink-0 overflow-hidden rounded-2xl bg-white/10 p-4 ring-1 ring-white/25 backdrop-blur-md lg:w-[340px]"
          >
            <div class="pointer-events-none absolute -right-8 -top-8 h-28 w-28 rounded-full bg-amber-300/30 blur-2xl" />
            <div class="relative flex items-center justify-between gap-2">
              <span class="inline-flex items-center gap-1 rounded-full bg-gradient-to-r from-amber-300 to-yellow-400 px-2.5 py-0.5 text-xs font-bold text-amber-900 shadow-lg shadow-amber-500/30">
                <Icon name="trophy" size="xs" :stroke-width="2" />
                {{ championTitle }}
              </span>
              <span v-if="champion.requests_growth !== null" class="rounded-full bg-white/15 px-2 py-0.5 text-xs font-semibold tabular-nums">
                {{ formatGrowth(champion.requests_growth) }}
              </span>
            </div>
            <div class="relative mt-3 flex items-center gap-3">
              <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-white shadow-lg">
                <ModelIcon :model="champion.model" size="26px" />
              </div>
              <p class="min-w-0 truncate text-lg font-bold" :title="champion.model">{{ champion.model }}</p>
            </div>
            <div class="relative mt-3 flex items-baseline gap-2">
              <span class="text-3xl font-black tabular-nums tracking-tight">{{ formatInt(champion.requests) }}</span>
              <span class="text-xs text-white/70">{{ t('modelLeaderboard.metric.requests') }}</span>
            </div>
            <div class="relative mt-2 h-1.5 w-full overflow-hidden rounded-full bg-white/20">
              <div class="h-full rounded-full bg-gradient-to-r from-amber-200 to-yellow-300" :style="{ width: pct(champion.request_share) }" />
            </div>
            <div class="relative mt-2 flex flex-wrap justify-between gap-x-3 gap-y-1 text-xs text-white/80">
              <span>{{ t('modelLeaderboard.hero.share', { pct: pct(champion.request_share) }) }}</span>
              <span v-if="runnerUp">{{ t('modelLeaderboard.hero.lead', { n: formatInt(champion.requests - runnerUp.requests) }) }}</span>
            </div>
          </div>
        </div>
      </section>

      <div v-if="loading && !data" class="flex items-center justify-center py-16"><LoadingSpinner /></div>
      <div v-else-if="loadFailed && !data" class="card p-8 text-center text-sm text-red-600 dark:text-red-400">
        {{ t('modelLeaderboard.loadFailed') }}
      </div>

      <template v-else-if="data">
        <!-- ============ 当月 / 历史累计 ============ -->
        <template v-if="activeTab !== 'months'">
          <!-- 汇总卡片 -->
          <div class="grid grid-cols-2 gap-4 lg:grid-cols-4" :class="data.cost_visible ? 'xl:grid-cols-6' : ''">
            <div
              v-for="card in summaryCards"
              :key="card.key"
              class="card relative overflow-hidden p-4 transition-all duration-200 hover:-translate-y-0.5 hover:shadow-card-hover"
            >
              <div class="flex items-start justify-between gap-2">
                <div class="min-w-0">
                  <p class="truncate text-xs font-medium text-gray-500 dark:text-dark-400">{{ card.label }}</p>
                  <p class="mt-1 truncate text-2xl font-bold tabular-nums text-gray-900 dark:text-white" :title="card.value">{{ card.value }}</p>
                </div>
                <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br text-white shadow-lg" :class="card.gradient">
                  <Icon :name="card.icon" size="md" />
                </div>
              </div>
              <div class="mt-2 flex h-5 min-w-0 items-center gap-1.5 text-xs">
                <template v-if="card.delta !== undefined">
                  <span class="shrink-0 rounded-full px-1.5 py-0.5 text-[11px] font-semibold tabular-nums" :class="deltaChipClass(card.delta)">
                    {{ formatGrowth(card.delta) }}
                  </span>
                  <span class="truncate text-gray-400 dark:text-dark-500">{{ card.deltaLabel }}</span>
                </template>
                <span v-else-if="card.hint" class="truncate text-gray-400 dark:text-dark-500">{{ card.hint }}</span>
              </div>
              <svg v-if="card.spark" class="mt-2 h-9 w-full" viewBox="0 0 120 32" preserveAspectRatio="none" aria-hidden="true">
                <defs>
                  <linearGradient :id="`lb-spark-${card.key}`" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="0%" :stop-color="card.color" stop-opacity="0.35" />
                    <stop offset="100%" :stop-color="card.color" stop-opacity="0" />
                  </linearGradient>
                </defs>
                <path :d="card.spark.area" :fill="`url(#lb-spark-${card.key})`" />
                <path
                  :d="card.spark.line"
                  fill="none"
                  :stroke="card.color"
                  stroke-width="1.75"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  vector-effect="non-scaling-stroke"
                />
              </svg>
              <div v-else class="mt-2 h-9" />
            </div>
          </div>

          <div v-if="!period!.ranking.length" class="card p-10 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelLeaderboard.empty') }}
          </div>

          <template v-else>
            <!-- 领奖台 Top 3：2-1-3 阶梯 -->
            <div class="grid grid-cols-1 items-end gap-4 md:grid-cols-3">
              <div
                v-for="item in podium"
                :key="item.model"
                class="flex flex-col"
                :class="medal(item.rank).order"
              >
                <div
                  class="card relative overflow-hidden p-5 transition-all duration-300 hover:-translate-y-1"
                  :class="medal(item.rank).card"
                >
                  <div class="absolute inset-x-0 top-0 h-1 bg-gradient-to-r" :class="medal(item.rank).bar" />
                  <div class="pointer-events-none absolute -right-12 -top-12 h-36 w-36 rounded-full blur-2xl" :class="medal(item.rank).glow" />

                  <div class="relative flex items-center gap-3">
                    <div
                      class="flex h-12 w-12 shrink-0 items-center justify-center rounded-full bg-gradient-to-br text-white shadow-lg ring-4"
                      :class="medal(item.rank).badge"
                    >
                      <Icon :name="item.rank === 1 ? 'trophy' : 'badge'" size="md" :stroke-width="2" />
                    </div>
                    <div class="min-w-0">
                      <p class="text-xs font-bold uppercase tracking-wider" :class="medal(item.rank).text">
                        {{ t(`modelLeaderboard.podium.${medal(item.rank).key}`) }}
                      </p>
                      <div class="mt-0.5 flex min-w-0 items-center gap-1.5">
                        <ModelIcon :model="item.model" size="18px" />
                        <p class="truncate text-base font-semibold text-gray-900 dark:text-white" :title="item.model">{{ item.model }}</p>
                      </div>
                    </div>
                  </div>

                  <p class="relative mt-4 text-3xl font-black tabular-nums tracking-tight text-gray-900 dark:text-white">
                    {{ formatInt(item.requests) }}
                    <span class="text-sm font-normal text-gray-500 dark:text-dark-400">{{ t('modelLeaderboard.metric.requests') }}</span>
                  </p>
                  <div class="relative mt-2 h-2 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                    <div class="h-full rounded-full bg-gradient-to-r" :class="medal(item.rank).bar" :style="{ width: pct(item.request_share) }" />
                  </div>
                  <div class="relative mt-3 grid grid-cols-3 gap-2 text-center">
                    <div class="rounded-lg bg-gray-50 px-2 py-1.5 dark:bg-dark-800/70">
                      <p class="text-[10px] text-gray-400 dark:text-dark-500">{{ t('modelLeaderboard.table.share') }}</p>
                      <p class="text-sm font-semibold tabular-nums text-gray-800 dark:text-dark-100">{{ pct(item.request_share) }}</p>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-2 py-1.5 dark:bg-dark-800/70">
                      <p class="text-[10px] text-gray-400 dark:text-dark-500">Tokens</p>
                      <p class="text-sm font-semibold tabular-nums text-gray-800 dark:text-dark-100">{{ formatCompactNumber(item.total_tokens) }}</p>
                    </div>
                    <div class="rounded-lg bg-gray-50 px-2 py-1.5 dark:bg-dark-800/70">
                      <p class="text-[10px] text-gray-400 dark:text-dark-500">{{ t('modelLeaderboard.table.users') }}</p>
                      <p class="text-sm font-semibold tabular-nums text-gray-800 dark:text-dark-100">{{ formatInt(item.users) }}</p>
                    </div>
                  </div>
                  <div class="relative mt-3 flex justify-end">
                    <RankMove :item="item" show-growth />
                  </div>
                </div>
                <div
                  class="mx-3 hidden items-center justify-center rounded-b-xl bg-gradient-to-b text-2xl font-black text-white/90 shadow-inner md:flex"
                  :class="medal(item.rank).pedestal"
                >
                  {{ item.rank }}
                </div>
              </div>
            </div>

            <!-- 图表区 -->
            <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div class="card p-5">
                <div class="mb-4 flex items-center gap-2">
                  <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300"><Icon name="chart" size="sm" /></span>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.share') }}</h3>
                </div>
                <div class="relative mx-auto h-48 max-w-[12rem]">
                  <Doughnut :data="shareChart" :options="doughnutOptions" />
                  <div class="pointer-events-none absolute inset-0 flex flex-col items-center justify-center">
                    <span class="text-xl font-black tabular-nums text-gray-900 dark:text-white">{{ formatCompactNumber(period!.summary.requests) }}</span>
                    <span class="text-[11px] text-gray-400 dark:text-dark-500">{{ t('modelLeaderboard.charts.totalRequests') }}</span>
                  </div>
                </div>
                <ul class="mt-4 space-y-1.5">
                  <li v-for="seg in shareLegend" :key="seg.label" class="flex items-center gap-2 text-xs">
                    <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ background: seg.color }" />
                    <span class="min-w-0 flex-1 truncate text-gray-600 dark:text-dark-300" :title="seg.label">{{ seg.label }}</span>
                    <span class="tabular-nums font-medium text-gray-900 dark:text-white">{{ pct(seg.share) }}</span>
                  </li>
                </ul>
              </div>
              <div class="card p-5 lg:col-span-2">
                <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                  <div class="flex items-center gap-2">
                    <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-indigo-100 text-indigo-600 dark:bg-indigo-900/40 dark:text-indigo-300"><Icon name="chartBar" size="sm" /></span>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.topBar') }}</h3>
                  </div>
                  <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
                    <button
                      v-for="m in barMetrics"
                      :key="m"
                      type="button"
                      class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
                      :class="barMetric === m ? activeSegClass : idleSegClass"
                      @click="barMetric = m"
                    >
                      {{ t(`modelLeaderboard.metric.${m}`) }}
                    </button>
                  </div>
                </div>
                <div class="h-80">
                  <Bar :data="topBarChart" :options="hBarOptions" />
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
              <div class="card p-5 xl:col-span-2">
                <div class="mb-4 flex flex-wrap items-center justify-between gap-2">
                  <div class="flex items-center gap-2">
                    <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-600 dark:bg-amber-900/40 dark:text-amber-300"><Icon name="trendingUp" size="sm" /></span>
                    <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                      {{ activeTab === 'monthly' ? t('modelLeaderboard.charts.trendDaily') : t('modelLeaderboard.charts.trendMonthly') }}
                    </h3>
                  </div>
                  <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
                    <button
                      v-for="m in trendMetrics"
                      :key="m"
                      type="button"
                      class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
                      :class="trendMetric === m ? activeSegClass : idleSegClass"
                      @click="trendMetric = m"
                    >
                      {{ t(`modelLeaderboard.metric.${m}`) }}
                    </button>
                  </div>
                </div>
                <div class="h-72">
                  <Bar :data="trendChart" :options="stackedOptions" />
                </div>
              </div>
              <div class="card p-5">
                <div class="mb-4 flex items-center gap-2">
                  <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-cyan-100 text-cyan-600 dark:bg-cyan-900/40 dark:text-cyan-300"><Icon name="database" size="sm" /></span>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.tokenMix') }}</h3>
                </div>
                <div class="h-72">
                  <Bar :data="tokenMixChart" :options="tokenMixOptions" />
                </div>
              </div>
            </div>

            <!-- 完整排行表 -->
            <div class="card overflow-hidden">
              <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-5 py-3.5 dark:border-dark-700">
                <div class="flex items-center gap-2">
                  <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-rose-100 text-rose-600 dark:bg-rose-900/40 dark:text-rose-300"><Icon name="fire" size="sm" /></span>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t('modelLeaderboard.table.title') }}
                    <span class="ml-1 text-xs font-normal text-gray-400">({{ filteredRanking.length }})</span>
                  </h3>
                </div>
                <div class="relative">
                  <Icon name="search" size="sm" class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
                  <input
                    v-model.trim="keyword"
                    type="search"
                    class="input h-8 w-52 py-0 pl-8 text-sm"
                    :placeholder="t('modelLeaderboard.table.search')"
                  />
                </div>
              </div>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[1000px] text-sm">
                  <thead class="bg-gray-50/80 text-xs text-gray-500 dark:bg-dark-800/80 dark:text-dark-400">
                    <tr>
                      <th
                        v-for="col in columns"
                        :key="col.key"
                        class="whitespace-nowrap px-3 py-2.5 font-medium"
                        :class="[col.align === 'left' ? 'text-left' : 'text-right', col.sortable ? 'cursor-pointer select-none hover:text-gray-900 dark:hover:text-white' : '']"
                        @click="col.sortable && toggleSort(col.key)"
                      >
                        {{ col.label }}
                        <span v-if="sortKey === col.key" class="text-primary-500">{{ sortDesc ? '↓' : '↑' }}</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                    <tr
                      v-for="row in filteredRanking"
                      :key="row.model"
                      class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/60"
                      :class="rowHighlight(row.rank)"
                    >
                      <td class="px-3 py-2.5 text-left tabular-nums">
                        <span
                          v-if="row.rank <= 3"
                          class="inline-flex h-7 w-7 items-center justify-center rounded-full bg-gradient-to-br text-xs font-black text-white shadow-md ring-2 ring-white dark:ring-dark-800"
                          :class="medal(row.rank).badge"
                        >{{ row.rank }}</span>
                        <span v-else class="inline-flex h-7 w-7 items-center justify-center text-xs font-semibold text-gray-400 dark:text-dark-500">{{ row.rank }}</span>
                      </td>
                      <td class="max-w-[260px] px-3 py-2.5 text-left">
                        <div class="flex items-center gap-2">
                          <ModelIcon :model="row.model" size="18px" />
                          <span class="truncate text-gray-900 dark:text-white" :class="row.rank <= 3 ? 'font-semibold' : 'font-medium'" :title="row.model">{{ row.model }}</span>
                        </div>
                      </td>
                      <td class="px-3 py-2.5 text-right font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatInt(row.requests) }}</td>
                      <td class="px-3 py-2.5 text-right">
                        <div class="flex items-center justify-end gap-2">
                          <div class="h-2 w-24 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                            <div class="h-full rounded-full" :style="{ width: pct(row.request_share), background: barGradient(row.model) }" />
                          </div>
                          <span class="w-12 tabular-nums text-gray-600 dark:text-dark-300">{{ pct(row.request_share) }}</span>
                        </div>
                      </td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-dark-300" :title="formatInt(row.total_tokens)">{{ formatCompactNumber(row.total_tokens) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatCompactNumber(row.input_tokens) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatCompactNumber(row.output_tokens) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ pct(row.cache_hit_rate) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatInt(row.users) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatMs(row.avg_duration_ms) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatMs(row.avg_first_token_ms) }}</td>
                      <td class="px-3 py-2.5 text-right"><RankMove :item="row" show-growth /></td>
                      <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-green-600 dark:text-green-400">${{ formatMoney(row.actual_cost) }}</td>
                      <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-gray-400 dark:text-dark-500">${{ formatMoney(row.cost) }}</td>
                      <td class="whitespace-nowrap px-3 py-2.5 text-right text-xs text-gray-500 dark:text-dark-400">{{ shortTime(row.last_used_at) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </template>
        </template>

        <!-- ============ 月度明细 ============ -->
        <template v-else>
          <div class="card p-5">
            <div class="mb-4 flex items-center gap-2">
              <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-900/40 dark:text-primary-300"><Icon name="trendingUp" size="sm" /></span>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.trendMonthly') }}</h3>
            </div>
            <div class="h-72">
              <Bar :data="monthTotalsChart" :options="monthTotalsOptions" />
            </div>
          </div>
          <div class="card overflow-hidden">
            <div class="flex items-center gap-2 border-b border-gray-100 px-5 py-3.5 dark:border-dark-700">
              <span class="flex h-7 w-7 items-center justify-center rounded-lg bg-amber-100 text-amber-600 dark:bg-amber-900/40 dark:text-amber-300"><Icon name="calendar" size="sm" /></span>
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.months.title') }}</h3>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full min-w-[820px] text-sm">
                <thead class="bg-gray-50/80 text-xs text-gray-500 dark:bg-dark-800/80 dark:text-dark-400">
                  <tr>
                    <th class="px-3 py-2.5 text-left font-medium">{{ t('modelLeaderboard.months.month') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.table.requests') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.table.growth') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.table.tokens') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.summary.users') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.summary.models') }}</th>
                    <th class="px-3 py-2.5 text-left font-medium">{{ t('modelLeaderboard.months.topModel') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.months.topShare') }}</th>
                    <th v-if="data.cost_visible" class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.table.actualCost') }}</th>
                    <th v-if="data.cost_visible" class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.table.cost') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr
                    v-for="(row, i) in data.month_rows"
                    :key="row.month"
                    class="group cursor-pointer transition-colors hover:bg-primary-50/60 dark:hover:bg-primary-900/10"
                    @click="openMonth(row.month)"
                  >
                    <td class="whitespace-nowrap px-3 py-2.5">
                      <span class="font-semibold text-primary-600 group-hover:underline dark:text-primary-400">{{ row.month }}</span>
                      <span
                        v-if="row.month === peakMonth"
                        class="ml-2 inline-flex items-center gap-0.5 rounded-full bg-gradient-to-r from-amber-400 to-orange-500 px-1.5 py-0.5 align-middle text-[10px] font-bold text-white shadow-sm shadow-orange-500/30"
                      >
                        <Icon name="fire" size="xs" :stroke-width="2" />{{ t('modelLeaderboard.months.peak') }}
                      </span>
                    </td>
                    <td class="px-3 py-2.5 text-right">
                      <div class="flex items-center justify-end gap-2">
                        <div class="hidden h-1.5 w-20 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700 sm:block">
                          <div class="h-full rounded-full bg-gradient-to-r from-primary-400 to-primary-600" :style="{ width: pct(monthMaxRequests ? row.requests / monthMaxRequests : 0) }" />
                        </div>
                        <span class="font-semibold tabular-nums text-gray-900 dark:text-white">{{ formatInt(row.requests) }}</span>
                      </div>
                    </td>
                    <td class="px-3 py-2.5 text-right">
                      <span class="rounded-full px-1.5 py-0.5 text-[11px] font-semibold tabular-nums" :class="deltaChipClass(monthGrowth(i))">{{ formatGrowth(monthGrowth(i)) }}</span>
                    </td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatCompactNumber(row.total_tokens) }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatInt(row.users) }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatInt(row.models) }}</td>
                    <td class="max-w-[220px] px-3 py-2.5">
                      <div class="flex items-center gap-2">
                        <ModelIcon v-if="row.top_model" :model="row.top_model" size="16px" />
                        <span class="truncate font-medium text-gray-900 dark:text-white" :title="row.top_model">{{ row.top_model || '-' }}</span>
                      </div>
                    </td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ pct(row.top_model_share) }}</td>
                    <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-green-600 dark:text-green-400">${{ formatMoney(row.actual_cost) }}</td>
                    <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-gray-400 dark:text-dark-500">${{ formatMoney(row.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>

        <p class="text-xs text-gray-400 dark:text-dark-500">
          {{ t('modelLeaderboard.footnote', { tz: data.timezone }) }} · {{ shortTime(data.generated_at) }}
        </p>
      </template>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  Chart as ChartJS,
  ArcElement,
  BarElement,
  CategoryScale,
  LinearScale,
  Tooltip,
  Legend
} from 'chart.js'
import { Bar, Doughnut } from 'vue-chartjs'
import AppLayout from '@/components/layout/AppLayout.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatCompactNumber } from '@/utils/format'
import {
  getModelLeaderboard,
  LEADERBOARD_OTHER,
  type LeaderboardRankItem,
  type LeaderboardSource,
  type ModelLeaderboardResponse
} from '@/api/modelLeaderboard'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

type Tab = 'monthly' | 'allTime' | 'months'
type Metric = 'requests' | 'tokens' | 'users'
type IconName = InstanceType<typeof Icon>['$props']['name']

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const activeSegClass = 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
const idleSegClass = 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'
const heroActiveClass = 'bg-white text-primary-700 shadow-md'
const heroIdleClass = 'text-white/80 hover:bg-white/10 hover:text-white'

const sources: LeaderboardSource[] = ['requested', 'upstream']
const barMetrics: Metric[] = ['requests', 'tokens', 'users']
const trendMetrics: Metric[] = ['requests', 'tokens']

const initialTab = (['monthly', 'allTime', 'months'] as Tab[]).includes(route.query.tab as Tab)
  ? (route.query.tab as Tab)
  : 'monthly'
const activeTab = ref<Tab>(initialTab)
const source = ref<LeaderboardSource>(route.query.source === 'upstream' ? 'upstream' : 'requested')
const month = ref<string>(typeof route.query.month === 'string' ? route.query.month : '')
const barMetric = ref<Metric>('requests')
const trendMetric = ref<Metric>('requests')
const keyword = ref('')
const sortKey = ref<string>('rank')
const sortDesc = ref(false)

const tabs = computed(() => [
  { value: 'monthly' as Tab, label: t('modelLeaderboard.tabs.monthly') },
  { value: 'allTime' as Tab, label: t('modelLeaderboard.tabs.allTime') },
  { value: 'months' as Tab, label: t('modelLeaderboard.tabs.months') }
])

const data = ref<ModelLeaderboardResponse | null>(null)
const loading = ref(false)
const loadFailed = ref(false)
let controller: AbortController | null = null

async function load(_force = false) {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  loadFailed.value = false
  try {
    const resp = await getModelLeaderboard(
      { source: source.value, month: month.value || undefined },
      { signal: controller.signal }
    )
    data.value = resp
    if (!month.value) month.value = resp.month
  } catch (e: any) {
    if (e?.name === 'CanceledError' || e?.code === 'ERR_CANCELED') return
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

function setSource(s: LeaderboardSource) {
  if (source.value === s) return
  source.value = s
  load()
}

function openMonth(m: string) {
  month.value = m
  activeTab.value = 'monthly'
  load()
}

watch([activeTab, source, month], () => {
  const query: Record<string, string> = { ...(route.query as Record<string, string>) }
  query.tab = activeTab.value
  query.source = source.value
  if (month.value) query.month = month.value
  router.replace({ query }).catch(() => {})
})

onMounted(() => load())

const monthOptions = computed(() => data.value?.months ?? [])
const period = computed(() => (activeTab.value === 'allTime' ? data.value?.all_time : data.value?.monthly))

// ---------- 格式化 ----------
const intFmt = computed(() => new Intl.NumberFormat(locale.value === 'zh' ? 'zh-CN' : 'en-US'))
function formatInt(n: number | null | undefined) {
  return intFmt.value.format(n ?? 0)
}
function pct(v: number | null | undefined) {
  const x = (v ?? 0) * 100
  return `${x >= 10 || x === 0 ? x.toFixed(1) : x.toFixed(2)}%`
}
function formatMoney(v: number | null | undefined) {
  const n = v ?? 0
  return n >= 100 ? n.toFixed(2) : n.toFixed(4)
}
function formatMs(v: number) {
  if (!v) return '-'
  return v >= 1000 ? `${(v / 1000).toFixed(1)}s` : `${Math.round(v)}ms`
}
function formatGrowth(g: number | null | undefined) {
  if (g === null || g === undefined || !isFinite(g)) return '-'
  if (g > 9.99) return '>+999%'
  const sign = g > 0 ? '+' : ''
  return `${sign}${(g * 100).toFixed(1)}%`
}
function deltaClass(g: number | null | undefined) {
  if (g === null || g === undefined || g === 0) return 'text-gray-400 dark:text-dark-500'
  return g > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-rose-600 dark:text-rose-400'
}
function deltaChipClass(g: number | null | undefined) {
  if (g === null || g === undefined || g === 0) return 'bg-gray-100 text-gray-500 dark:bg-dark-700 dark:text-dark-400'
  return g > 0
    ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-500/10 dark:text-emerald-400'
    : 'bg-rose-50 text-rose-600 dark:bg-rose-500/10 dark:text-rose-400'
}
function shortTime(iso: string | undefined) {
  if (!iso) return '-'
  const d = new Date(iso)
  if (isNaN(d.getTime())) return iso
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}
function growthOf(cur: number, prev: number | undefined): number | null {
  if (prev === undefined || prev <= 0) return null
  return (cur - prev) / prev
}

// ---------- Hero ----------
const heroPeriod = computed(() => (activeTab.value === 'months' ? data.value?.all_time : period.value))
const champion = computed(() => heroPeriod.value?.ranking[0])
const runnerUp = computed(() => heroPeriod.value?.ranking[1])
const championTitle = computed(() => {
  if (activeTab.value !== 'monthly') return t('modelLeaderboard.hero.allTimeChampion')
  return data.value?.is_current
    ? t('modelLeaderboard.hero.currentChampion')
    : t('modelLeaderboard.hero.monthChampion', { month: data.value?.month ?? '' })
})
const periodCaption = computed(() => {
  const p = heroPeriod.value
  if (!p) return ''
  const start = (p.summary.first_at || p.start || '').slice(0, 10)
  const end = (activeTab.value === 'monthly' ? p.end : p.summary.last_at || p.end || '').slice(0, 10)
  return start && end ? t('modelLeaderboard.summary.range', { start, end }) : ''
})

// ---------- 汇总卡片 ----------
function sparkPaths(values: number[] | undefined, w = 120, hgt = 32) {
  if (!values || values.length < 2 || values.every((v) => v === 0)) return undefined
  const max = Math.max(...values)
  const min = Math.min(...values)
  const span = max - min || 1
  const step = w / (values.length - 1)
  const pts = values.map((v, i) => [i * step, hgt - 2 - ((v - min) / span) * (hgt - 6)])
  const line = pts.map(([x, y], i) => `${i ? 'L' : 'M'}${x.toFixed(1)},${y.toFixed(1)}`).join(' ')
  return { line, area: `${line} L${w},${hgt} L0,${hgt} Z` }
}

const trendTotals = computed(() => {
  const s = period.value?.trend
  if (!s) return { requests: [] as number[], tokens: [] as number[] }
  const sum = (key: 'requests' | 'tokens') =>
    (s.labels ?? []).map((_, i) => (s.datasets ?? []).reduce((a, d) => a + (d[key][i] ?? 0), 0))
  return { requests: sum('requests'), tokens: sum('tokens') }
})

const monthSeries = computed(() => [...(data.value?.month_rows ?? [])].reverse())

type SummaryCard = {
  key: string
  label: string
  value: string
  icon: IconName
  gradient: string
  color: string
  delta?: number | null
  deltaLabel?: string
  hint?: string
  spark?: { line: string; area: string }
}

const summaryCards = computed<SummaryCard[]>(() => {
  const p = period.value
  if (!p || !data.value) return []
  const s = p.summary
  const allTime = activeTab.value === 'allTime'
  const prev = activeTab.value === 'monthly' ? p.prev : undefined
  const deltaLabel = data.value.is_current ? t('modelLeaderboard.summary.vsPrev') : t('modelLeaderboard.summary.vsPrevMonth')
  const hint = allTime && s.first_at
    ? t('modelLeaderboard.summary.since', { date: s.first_at.slice(0, 10) })
    : undefined
  const ms = monthSeries.value
  const cards: SummaryCard[] = [
    {
      key: 'req', label: t('modelLeaderboard.summary.requests'), value: formatInt(s.requests),
      icon: 'chart', gradient: 'from-primary-500 to-primary-600 shadow-primary-500/30', color: '#14b8a6',
      delta: prev ? growthOf(s.requests, prev.requests) : undefined, deltaLabel, hint,
      spark: sparkPaths(trendTotals.value.requests)
    },
    {
      key: 'tok', label: t('modelLeaderboard.summary.tokens'), value: formatCompactNumber(s.total_tokens),
      icon: 'database', gradient: 'from-indigo-500 to-violet-600 shadow-indigo-500/30', color: '#6366f1',
      delta: prev ? growthOf(s.total_tokens, prev.total_tokens) : undefined, deltaLabel, hint,
      spark: sparkPaths(trendTotals.value.tokens)
    },
    {
      key: 'usr', label: t('modelLeaderboard.summary.users'), value: formatInt(s.users),
      icon: 'users', gradient: 'from-amber-400 to-orange-500 shadow-orange-500/30', color: '#f59e0b',
      delta: prev ? growthOf(s.users, prev.users) : undefined, deltaLabel, hint,
      spark: allTime ? sparkPaths(ms.map((r) => r.users)) : undefined
    },
    {
      key: 'mdl', label: t('modelLeaderboard.summary.models'), value: formatInt(s.models),
      icon: 'cube', gradient: 'from-pink-500 to-rose-500 shadow-rose-500/30', color: '#ec4899',
      delta: prev ? growthOf(s.models, prev.models) : undefined, deltaLabel, hint,
      spark: allTime ? sparkPaths(ms.map((r) => r.models)) : undefined
    }
  ]
  if (data.value.cost_visible) {
    cards.push(
      {
        key: 'act', label: t('modelLeaderboard.summary.actualCost'), value: `$${formatMoney(s.actual_cost)}`,
        icon: 'dollar', gradient: 'from-emerald-500 to-green-600 shadow-emerald-500/30', color: '#10b981',
        delta: prev ? growthOf(s.actual_cost ?? 0, prev.actual_cost) : undefined, deltaLabel, hint,
        spark: allTime ? sparkPaths(ms.map((r) => r.actual_cost ?? 0)) : undefined
      },
      {
        key: 'std', label: t('modelLeaderboard.summary.cost'), value: `$${formatMoney(s.cost)}`,
        icon: 'creditCard', gradient: 'from-sky-500 to-blue-600 shadow-blue-500/30', color: '#3b82f6',
        delta: prev ? growthOf(s.cost ?? 0, prev.cost) : undefined, deltaLabel, hint,
        spark: allTime ? sparkPaths(ms.map((r) => r.cost ?? 0)) : undefined
      }
    )
  }
  return cards
})

// ---------- 颜色 ----------
const palette = ['#14b8a6', '#6366f1', '#f59e0b', '#ec4899', '#3b82f6', '#8b5cf6', '#10b981', '#f97316', '#06b6d4', '#ef4444', '#84cc16', '#a855f7']
const otherColor = '#94a3b8'
const colorMap = computed(() => {
  const m = new Map<string, string>()
  const order = [...(data.value?.all_time.ranking ?? []), ...(data.value?.monthly.ranking ?? [])]
  for (const r of order) if (!m.has(r.model)) m.set(r.model, palette[m.size % palette.length])
  m.set(LEADERBOARD_OTHER, otherColor)
  return m
})
function colorOf(model: string) {
  return colorMap.value.get(model) ?? otherColor
}
function barGradient(model: string) {
  const c = colorOf(model)
  return `linear-gradient(90deg, ${c}80, ${c})`
}
function labelOf(model: string) {
  return model === LEADERBOARD_OTHER ? t('modelLeaderboard.charts.other') : model
}

const medalStyles = {
  1: {
    key: 'first',
    order: 'md:order-2',
    card: 'border-amber-200/80 shadow-xl shadow-amber-500/10 dark:border-amber-500/30',
    bar: 'from-amber-300 via-yellow-400 to-amber-500',
    glow: 'bg-amber-400/25',
    badge: 'from-amber-300 via-yellow-400 to-amber-500 shadow-amber-500/40 ring-amber-100 dark:ring-amber-500/20',
    text: 'text-amber-600 dark:text-amber-400',
    pedestal: 'h-20 from-amber-400 to-amber-600'
  },
  2: {
    key: 'second',
    order: 'md:order-1',
    card: 'border-slate-200 dark:border-slate-500/30',
    bar: 'from-slate-300 via-gray-300 to-slate-400',
    glow: 'bg-slate-400/20',
    badge: 'from-slate-300 via-gray-400 to-slate-500 shadow-slate-500/40 ring-slate-100 dark:ring-slate-500/20',
    text: 'text-slate-500 dark:text-slate-300',
    pedestal: 'h-14 from-slate-300 to-slate-500'
  },
  3: {
    key: 'third',
    order: 'md:order-3',
    card: 'border-orange-200/80 dark:border-orange-500/30',
    bar: 'from-orange-300 via-amber-600 to-orange-700',
    glow: 'bg-orange-400/20',
    badge: 'from-orange-300 via-orange-500 to-amber-700 shadow-orange-500/40 ring-orange-100 dark:ring-orange-500/20',
    text: 'text-orange-600 dark:text-orange-400',
    pedestal: 'h-10 from-orange-400 to-amber-700'
  }
} as const

function medal(rank: number) {
  return medalStyles[(rank >= 1 && rank <= 3 ? rank : 3) as 1 | 2 | 3]
}
function rowHighlight(rank: number) {
  if (rank === 1) return 'bg-gradient-to-r from-amber-50 via-amber-50/40 to-transparent dark:from-amber-500/10 dark:via-amber-500/5'
  if (rank === 2) return 'bg-gradient-to-r from-slate-100/80 via-slate-50/40 to-transparent dark:from-slate-400/10 dark:via-slate-400/5'
  if (rank === 3) return 'bg-gradient-to-r from-orange-50 via-orange-50/40 to-transparent dark:from-orange-500/10 dark:via-orange-500/5'
  return ''
}

const isDark = () => document.documentElement.classList.contains('dark')
const axisColor = () => (isDark() ? '#9ca3af' : '#6b7280')
const gridColor = () => (isDark() ? 'rgba(75,85,99,0.35)' : 'rgba(229,231,235,0.8)')
const tooltipStyle = () => ({
  backgroundColor: isDark() ? 'rgba(15,23,42,0.95)' : 'rgba(255,255,255,0.97)',
  titleColor: isDark() ? '#f1f5f9' : '#0f172a',
  bodyColor: isDark() ? '#cbd5e1' : '#334155',
  borderColor: isDark() ? 'rgba(71,85,105,0.6)' : 'rgba(226,232,240,1)',
  borderWidth: 1,
  padding: 10,
  cornerRadius: 10,
  boxPadding: 4,
  usePointStyle: true
})

// ---------- 图表 ----------
const podium = computed(() => (period.value?.ranking ?? []).slice(0, 3))

const shareChart = computed(() => {
  const rows = period.value?.ranking ?? []
  const top = rows.slice(0, 8)
  const rest = rows.slice(8).reduce((a, r) => a + r.requests, 0)
  const labels = top.map((r) => r.model)
  const values = top.map((r) => r.requests)
  const colors = top.map((r) => colorOf(r.model))
  if (rest > 0) {
    labels.push(t('modelLeaderboard.charts.other'))
    values.push(rest)
    colors.push(otherColor)
  }
  return {
    labels,
    datasets: [{
      data: values,
      backgroundColor: colors,
      borderWidth: 2,
      borderColor: isDark() ? '#1e293b' : '#ffffff',
      hoverOffset: 6,
      borderRadius: 4
    }]
  }
})

const shareLegend = computed(() => {
  const c = shareChart.value
  const values = c.datasets[0].data
  const total = values.reduce((a, b) => a + b, 0)
  return c.labels.slice(0, 6).map((label, i) => ({
    label,
    color: c.datasets[0].backgroundColor[i],
    share: total ? values[i] / total : 0
  }))
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '70%',
  plugins: {
    legend: { display: false },
    tooltip: {
      ...tooltipStyle(),
      callbacks: {
        label: (ctx: any) => {
          const total = (ctx.dataset.data as number[]).reduce((a, b) => a + b, 0)
          return `${ctx.label}: ${formatInt(ctx.raw)} (${pct(total ? ctx.raw / total : 0)})`
        }
      }
    }
  }
}))

function metricValue(r: LeaderboardRankItem, m: Metric) {
  return m === 'tokens' ? r.total_tokens : m === 'users' ? r.users : r.requests
}

const topBarChart = computed(() => {
  const rows = [...(period.value?.ranking ?? [])]
    .sort((a, b) => metricValue(b, barMetric.value) - metricValue(a, barMetric.value))
    .slice(0, 10)
  return {
    labels: rows.map((r) => r.model),
    datasets: [{
      label: t(`modelLeaderboard.metric.${barMetric.value}`),
      data: rows.map((r) => metricValue(r, barMetric.value)),
      backgroundColor: rows.map((r) => colorOf(r.model)),
      borderRadius: 6,
      borderSkipped: false,
      maxBarThickness: 20
    }]
  }
})

const hBarOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: { ...tooltipStyle(), callbacks: { label: (ctx: any) => formatInt(ctx.raw) } }
  },
  scales: {
    x: { ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() }, border: { display: false } },
    y: { ticks: { color: axisColor(), font: { size: 11 } }, grid: { display: false }, border: { display: false } }
  }
}))

const trendChart = computed(() => {
  const s = period.value?.trend
  if (!s) return { labels: [], datasets: [] }
  const raw = s.labels ?? []
  const labels = activeTab.value === 'monthly' ? raw.map((l) => l.slice(5)) : raw
  return {
    labels,
    datasets: (s.datasets ?? []).map((d) => ({
      label: labelOf(d.model),
      data: trendMetric.value === 'tokens' ? d.tokens : d.requests,
      backgroundColor: colorOf(d.model),
      stack: 'm',
      borderRadius: 3,
      maxBarThickness: 36
    }))
  }
})

const stackedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 8, boxHeight: 8, usePointStyle: true, font: { size: 11 } } },
    tooltip: {
      ...tooltipStyle(),
      itemSort: (a: any, b: any) => b.raw - a.raw,
      filter: (item: any) => item.raw > 0,
      callbacks: { label: (ctx: any) => `${ctx.dataset.label}: ${formatInt(ctx.raw)}` }
    }
  },
  scales: {
    x: { stacked: true, ticks: { color: axisColor(), maxRotation: 0, autoSkip: true }, grid: { display: false }, border: { display: false } },
    y: { stacked: true, ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() }, border: { display: false } }
  }
}))

const tokenMixChart = computed(() => {
  const rows = [...(period.value?.ranking ?? [])].sort((a, b) => b.total_tokens - a.total_tokens).slice(0, 10)
  const mk = (key: keyof LeaderboardRankItem, label: string, color: string) => ({
    label,
    data: rows.map((r) => {
      const total = r.total_tokens || 1
      return ((r[key] as number) / total) * 100
    }),
    raw: rows.map((r) => r[key] as number),
    backgroundColor: color,
    stack: 't',
    maxBarThickness: 16
  })
  return {
    labels: rows.map((r) => r.model),
    datasets: [
      mk('input_tokens', t('modelLeaderboard.charts.input'), '#6366f1'),
      mk('output_tokens', t('modelLeaderboard.charts.output'), '#14b8a6'),
      mk('cache_creation_tokens', t('modelLeaderboard.charts.cacheCreate'), '#f59e0b'),
      mk('cache_read_tokens', t('modelLeaderboard.charts.cacheRead'), '#38bdf8')
    ]
  }
})

const tokenMixOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 8, boxHeight: 8, usePointStyle: true, font: { size: 11 } } },
    tooltip: {
      ...tooltipStyle(),
      callbacks: {
        label: (ctx: any) => `${ctx.dataset.label}: ${ctx.raw.toFixed(1)}% (${formatCompactNumber(ctx.dataset.raw[ctx.dataIndex])})`
      }
    }
  },
  scales: {
    x: { stacked: true, max: 100, ticks: { color: axisColor(), callback: (v: any) => `${v}%` }, grid: { color: gridColor() }, border: { display: false } },
    y: { stacked: true, ticks: { color: axisColor(), font: { size: 10 } }, grid: { display: false }, border: { display: false } }
  }
}))

// 月度明细：调用量柱 + 活跃用户（右轴）
const monthTotalsChart = computed(() => {
  const rows = monthSeries.value
  return {
    labels: rows.map((r) => r.month),
    datasets: [
      { type: 'bar' as const, label: t('modelLeaderboard.metric.requests'), data: rows.map((r) => r.requests), backgroundColor: '#14b8a6', borderRadius: 6, borderSkipped: false, yAxisID: 'y', maxBarThickness: 44 },
      { type: 'bar' as const, label: t('modelLeaderboard.metric.users'), data: rows.map((r) => r.users), backgroundColor: '#f59e0b', borderRadius: 6, borderSkipped: false, yAxisID: 'y1', maxBarThickness: 44 }
    ]
  }
})

const monthTotalsOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 8, boxHeight: 8, usePointStyle: true } },
    tooltip: { ...tooltipStyle(), callbacks: { label: (ctx: any) => `${ctx.dataset.label}: ${formatInt(ctx.raw)}` } }
  },
  scales: {
    x: { ticks: { color: axisColor() }, grid: { display: false }, border: { display: false } },
    y: { position: 'left' as const, ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() }, border: { display: false } },
    y1: { position: 'right' as const, ticks: { color: axisColor(), precision: 0 }, grid: { display: false }, border: { display: false } }
  }
}))

const monthMaxRequests = computed(() => Math.max(0, ...(data.value?.month_rows ?? []).map((r) => r.requests)))
const peakMonth = computed(() => {
  const rows = data.value?.month_rows ?? []
  if (rows.length < 2 || !monthMaxRequests.value) return ''
  return rows.find((r) => r.requests === monthMaxRequests.value)?.month ?? ''
})

function monthGrowth(i: number): number | null {
  const rows = data.value?.month_rows ?? []
  const prev = rows[i + 1]
  return prev ? growthOf(rows[i].requests, prev.requests) : null
}

// ---------- 表格 ----------
const columns = computed(() => {
  const cols = [
    { key: 'rank', label: t('modelLeaderboard.table.rank'), align: 'left', sortable: true },
    { key: 'model', label: t('modelLeaderboard.table.model'), align: 'left', sortable: true },
    { key: 'requests', label: t('modelLeaderboard.table.requests'), sortable: true },
    { key: 'request_share', label: t('modelLeaderboard.table.share'), sortable: true },
    { key: 'total_tokens', label: t('modelLeaderboard.table.tokens'), sortable: true },
    { key: 'input_tokens', label: t('modelLeaderboard.table.input'), sortable: true },
    { key: 'output_tokens', label: t('modelLeaderboard.table.output'), sortable: true },
    { key: 'cache_hit_rate', label: t('modelLeaderboard.table.cacheHit'), sortable: true },
    { key: 'users', label: t('modelLeaderboard.table.users'), sortable: true },
    { key: 'avg_duration_ms', label: t('modelLeaderboard.table.latency'), sortable: true },
    { key: 'avg_first_token_ms', label: t('modelLeaderboard.table.ttft'), sortable: true },
    { key: 'requests_growth', label: t('modelLeaderboard.table.growth'), sortable: true }
  ] as Array<{ key: string; label: string; align?: string; sortable?: boolean }>
  if (data.value?.cost_visible) {
    cols.push(
      { key: 'actual_cost', label: t('modelLeaderboard.table.actualCost'), sortable: true },
      { key: 'cost', label: t('modelLeaderboard.table.cost'), sortable: true }
    )
  }
  cols.push({ key: 'last_used_at', label: t('modelLeaderboard.table.lastUsed'), sortable: true })
  return cols
})

function toggleSort(key: string) {
  if (sortKey.value === key) {
    sortDesc.value = !sortDesc.value
  } else {
    sortKey.value = key
    sortDesc.value = !['rank', 'model'].includes(key)
  }
}

const filteredRanking = computed(() => {
  const kw = keyword.value.toLowerCase()
  const rows = (period.value?.ranking ?? []).filter((r) => !kw || r.model.toLowerCase().includes(kw))
  const key = sortKey.value as keyof LeaderboardRankItem
  const dir = sortDesc.value ? -1 : 1
  return [...rows].sort((a, b) => {
    const av = a[key]
    const bv = b[key]
    if (av === bv) return a.rank - b.rank
    if (av === null || av === undefined) return 1
    if (bv === null || bv === undefined) return -1
    if (typeof av === 'string' && typeof bv === 'string') return av.localeCompare(bv) * dir
    return ((av as number) - (bv as number)) * dir
  })
})

// ---------- 排名变化小组件 ----------
const RankMove = defineComponent({
  props: {
    item: { type: Object as () => LeaderboardRankItem, required: true },
    showGrowth: { type: Boolean, default: false }
  },
  setup(props) {
    return () => {
      const it = props.item
      const parts = []
      if (it.prev_rank === null || it.prev_rank === undefined) {
        parts.push(h('span', { class: 'rounded-full bg-gradient-to-r from-primary-500 to-cyan-500 px-2 py-0.5 text-[10px] font-bold text-white shadow-sm shadow-primary-500/30' }, t('modelLeaderboard.table.newEntry')))
      } else {
        const diff = it.prev_rank - it.rank
        const cls = diff > 0 ? 'text-emerald-600 dark:text-emerald-400' : diff < 0 ? 'text-rose-600 dark:text-rose-400' : 'text-gray-400 dark:text-dark-500'
        parts.push(h('span', { class: `text-xs font-semibold ${cls}`, title: `#${it.prev_rank} → #${it.rank}` }, diff > 0 ? `▲${diff}` : diff < 0 ? `▼${-diff}` : '—'))
        if (props.showGrowth && it.requests_growth !== null && it.requests_growth !== undefined) {
          parts.push(h('span', { class: `ml-1.5 text-xs ${deltaClass(it.requests_growth)}` }, formatGrowth(it.requests_growth)))
        }
      }
      return h('span', { class: 'inline-flex items-center whitespace-nowrap' }, parts)
    }
  }
})
</script>
