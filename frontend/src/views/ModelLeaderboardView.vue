<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 工具条：口径 / 标签 / 月份 / 刷新 -->
      <div class="card flex flex-wrap items-center gap-3 p-4">
        <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800" role="tablist">
          <button
            v-for="tab in tabs"
            :key="tab.value"
            type="button"
            role="tab"
            :aria-selected="activeTab === tab.value"
            class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
            :class="activeTab === tab.value ? activeSegClass : idleSegClass"
            @click="activeTab = tab.value"
          >
            {{ tab.label }}
          </button>
        </div>

        <div class="flex items-center gap-2">
          <span class="text-sm text-gray-500 dark:text-dark-400">{{ t('modelLeaderboard.source.label') }}</span>
          <div class="inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800">
            <button
              v-for="s in sources"
              :key="s"
              type="button"
              class="rounded-md px-2.5 py-1 text-xs font-medium transition-colors"
              :class="source === s ? activeSegClass : idleSegClass"
              @click="setSource(s)"
            >
              {{ t(`modelLeaderboard.source.${s}`) }}
            </button>
          </div>
        </div>

        <div v-if="activeTab === 'monthly' && monthOptions.length" class="flex items-center gap-2">
          <label for="lb-month" class="text-sm text-gray-500 dark:text-dark-400">{{ t('modelLeaderboard.month') }}</label>
          <select
            id="lb-month"
            v-model="month"
            class="input h-8 w-32 py-0 text-sm"
            @change="load()"
          >
            <option v-for="m in monthOptions" :key="m" :value="m">{{ m }}</option>
          </select>
        </div>

        <button type="button" class="btn btn-secondary ml-auto" :disabled="loading" @click="load(true)">
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span class="ml-1.5">{{ t('modelLeaderboard.refresh') }}</span>
        </button>
      </div>

      <div v-if="loading && !data" class="flex items-center justify-center py-16"><LoadingSpinner /></div>
      <div v-else-if="loadFailed && !data" class="card p-8 text-center text-sm text-red-600 dark:text-red-400">
        {{ t('modelLeaderboard.loadFailed') }}
      </div>

      <template v-else-if="data">
        <!-- ============ 当月 / 历史累计 ============ -->
        <template v-if="activeTab !== 'months'">
          <!-- 汇总卡片 -->
          <div class="grid grid-cols-2 gap-4 lg:grid-cols-4" :class="data.cost_visible ? 'xl:grid-cols-6' : ''">
            <div v-for="card in summaryCards" :key="card.key" class="card p-4">
              <p class="text-xs text-gray-500 dark:text-dark-400">{{ card.label }}</p>
              <p class="mt-1 text-2xl font-bold tabular-nums text-gray-900 dark:text-white">{{ card.value }}</p>
              <p v-if="card.delta !== undefined" class="mt-1 text-xs" :class="deltaClass(card.delta)">
                {{ formatGrowth(card.delta) }}
                <span class="text-gray-400 dark:text-dark-500">{{ card.deltaLabel }}</span>
              </p>
              <p v-else-if="card.hint" class="mt-1 truncate text-xs text-gray-400 dark:text-dark-500">{{ card.hint }}</p>
            </div>
          </div>

          <div v-if="!period!.ranking.length" class="card p-10 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelLeaderboard.empty') }}
          </div>

          <template v-else>
            <!-- 领奖台 Top 3 -->
            <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div
                v-for="item in podium"
                :key="item.model"
                class="card relative overflow-hidden p-5"
              >
                <div class="absolute right-3 top-3 text-3xl font-black tabular-nums" :class="podiumColor(item.rank)">#{{ item.rank }}</div>
                <div class="flex items-center gap-2">
                  <ModelIcon :model="item.model" size="22px" />
                  <p class="truncate pr-10 text-base font-semibold text-gray-900 dark:text-white" :title="item.model">{{ item.model }}</p>
                </div>
                <p class="mt-3 text-2xl font-bold tabular-nums text-gray-900 dark:text-white">
                  {{ formatInt(item.requests) }}
                  <span class="text-sm font-normal text-gray-500 dark:text-dark-400">{{ t('modelLeaderboard.metric.requests') }}</span>
                </p>
                <div class="mt-2 h-1.5 w-full rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-1.5 rounded-full" :style="{ width: pct(item.request_share), background: colorOf(item.model) }" />
                </div>
                <div class="mt-2 flex justify-between text-xs text-gray-500 dark:text-dark-400">
                  <span>{{ pct(item.request_share) }} · {{ formatCompactNumber(item.total_tokens) }} tokens</span>
                  <RankMove :item="item" />
                </div>
              </div>
            </div>

            <!-- 图表区 -->
            <div class="grid grid-cols-1 gap-6 lg:grid-cols-3">
              <div class="card p-4">
                <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.share') }}</h3>
                <div class="mx-auto h-64 max-w-xs">
                  <Doughnut :data="shareChart" :options="doughnutOptions" />
                </div>
              </div>
              <div class="card p-4 lg:col-span-2">
                <div class="mb-3 flex items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.topBar') }}</h3>
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
                <div class="h-64">
                  <Bar :data="topBarChart" :options="hBarOptions" />
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
              <div class="card p-4 xl:col-span-2">
                <div class="mb-3 flex items-center justify-between gap-2">
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ activeTab === 'monthly' ? t('modelLeaderboard.charts.trendDaily') : t('modelLeaderboard.charts.trendMonthly') }}
                  </h3>
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
              <div class="card p-4">
                <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.tokenMix') }}</h3>
                <div class="h-72">
                  <Bar :data="tokenMixChart" :options="tokenMixOptions" />
                </div>
              </div>
            </div>

            <!-- 完整排行表 -->
            <div class="card overflow-hidden">
              <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('modelLeaderboard.table.title') }}
                  <span class="ml-1 text-xs font-normal text-gray-400">({{ filteredRanking.length }})</span>
                </h3>
                <input
                  v-model.trim="keyword"
                  type="search"
                  class="input h-8 w-48 py-0 text-sm"
                  :placeholder="t('modelLeaderboard.table.search')"
                />
              </div>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[960px] text-sm">
                  <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                    <tr>
                      <th
                        v-for="col in columns"
                        :key="col.key"
                        class="whitespace-nowrap px-3 py-2 font-medium"
                        :class="[col.align === 'left' ? 'text-left' : 'text-right', col.sortable ? 'cursor-pointer select-none hover:text-gray-900 dark:hover:text-white' : '']"
                        @click="col.sortable && toggleSort(col.key)"
                      >
                        {{ col.label }}
                        <span v-if="sortKey === col.key">{{ sortDesc ? '↓' : '↑' }}</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                    <tr v-for="row in filteredRanking" :key="row.model" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                      <td class="px-3 py-2 text-left tabular-nums">
                        <span class="inline-flex h-6 min-w-6 items-center justify-center rounded-md px-1.5 text-xs font-bold" :class="rankBadge(row.rank)">{{ row.rank }}</span>
                      </td>
                      <td class="max-w-[260px] px-3 py-2 text-left">
                        <div class="flex items-center gap-2">
                          <ModelIcon :model="row.model" size="16px" />
                          <span class="truncate font-medium text-gray-900 dark:text-white" :title="row.model">{{ row.model }}</span>
                        </div>
                      </td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-900 dark:text-white">{{ formatInt(row.requests) }}</td>
                      <td class="px-3 py-2 text-right">
                        <div class="flex items-center justify-end gap-2">
                          <div class="h-1.5 w-16 rounded-full bg-gray-100 dark:bg-dark-700">
                            <div class="h-1.5 rounded-full" :style="{ width: pct(row.request_share), background: colorOf(row.model) }" />
                          </div>
                          <span class="w-12 tabular-nums text-gray-600 dark:text-dark-300">{{ pct(row.request_share) }}</span>
                        </div>
                      </td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-600 dark:text-dark-300" :title="formatInt(row.total_tokens)">{{ formatCompactNumber(row.total_tokens) }}</td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatCompactNumber(row.input_tokens) }}</td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatCompactNumber(row.output_tokens) }}</td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ pct(row.cache_hit_rate) }}</td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatInt(row.users) }}</td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatMs(row.avg_duration_ms) }}</td>
                      <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ formatMs(row.avg_first_token_ms) }}</td>
                      <td class="px-3 py-2 text-right"><RankMove :item="row" show-growth /></td>
                      <td v-if="data.cost_visible" class="px-3 py-2 text-right tabular-nums text-green-600 dark:text-green-400">${{ formatMoney(row.actual_cost) }}</td>
                      <td v-if="data.cost_visible" class="px-3 py-2 text-right tabular-nums text-gray-400 dark:text-dark-500">${{ formatMoney(row.cost) }}</td>
                      <td class="whitespace-nowrap px-3 py-2 text-right text-xs text-gray-500 dark:text-dark-400">{{ shortTime(row.last_used_at) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </template>
        </template>

        <!-- ============ 月度明细 ============ -->
        <template v-else>
          <div class="card p-4">
            <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.trendMonthly') }}</h3>
            <div class="h-72">
              <Bar :data="monthTotalsChart" :options="monthTotalsOptions" />
            </div>
          </div>
          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.months.title') }}</h3>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full min-w-[720px] text-sm">
                <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
                  <tr>
                    <th class="px-3 py-2 text-left font-medium">{{ t('modelLeaderboard.months.month') }}</th>
                    <th class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.table.requests') }}</th>
                    <th class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.table.growth') }}</th>
                    <th class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.table.tokens') }}</th>
                    <th class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.summary.users') }}</th>
                    <th class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.summary.models') }}</th>
                    <th class="px-3 py-2 text-left font-medium">{{ t('modelLeaderboard.months.topModel') }}</th>
                    <th class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.months.topShare') }}</th>
                    <th v-if="data.cost_visible" class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.table.actualCost') }}</th>
                    <th v-if="data.cost_visible" class="px-3 py-2 text-right font-medium">{{ t('modelLeaderboard.table.cost') }}</th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                  <tr
                    v-for="(row, i) in data.month_rows"
                    :key="row.month"
                    class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800/60"
                    @click="openMonth(row.month)"
                  >
                    <td class="px-3 py-2 font-medium text-primary-600 dark:text-primary-400">{{ row.month }}</td>
                    <td class="px-3 py-2 text-right tabular-nums text-gray-900 dark:text-white">{{ formatInt(row.requests) }}</td>
                    <td class="px-3 py-2 text-right text-xs" :class="deltaClass(monthGrowth(i))">{{ formatGrowth(monthGrowth(i)) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatCompactNumber(row.total_tokens) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatInt(row.users) }}</td>
                    <td class="px-3 py-2 text-right tabular-nums text-gray-600 dark:text-dark-300">{{ formatInt(row.models) }}</td>
                    <td class="max-w-[220px] px-3 py-2">
                      <div class="flex items-center gap-2">
                        <ModelIcon v-if="row.top_model" :model="row.top_model" size="16px" />
                        <span class="truncate text-gray-900 dark:text-white" :title="row.top_model">{{ row.top_model || '-' }}</span>
                      </div>
                    </td>
                    <td class="px-3 py-2 text-right tabular-nums text-gray-500 dark:text-dark-400">{{ pct(row.top_model_share) }}</td>
                    <td v-if="data.cost_visible" class="px-3 py-2 text-right tabular-nums text-green-600 dark:text-green-400">${{ formatMoney(row.actual_cost) }}</td>
                    <td v-if="data.cost_visible" class="px-3 py-2 text-right tabular-nums text-gray-400 dark:text-dark-500">${{ formatMoney(row.cost) }}</td>
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

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const activeSegClass = 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
const idleSegClass = 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'

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

// ---------- 汇总卡片 ----------
const summaryCards = computed(() => {
  const p = period.value
  if (!p || !data.value) return []
  const s = p.summary
  const prev = activeTab.value === 'monthly' ? p.prev : undefined
  const deltaLabel = data.value.is_current ? t('modelLeaderboard.summary.vsPrev') : t('modelLeaderboard.summary.vsPrevMonth')
  const hint = activeTab.value === 'allTime' && s.first_at
    ? t('modelLeaderboard.summary.since', { date: s.first_at.slice(0, 10) })
    : undefined
  const cards: Array<{ key: string; label: string; value: string; delta?: number | null; deltaLabel?: string; hint?: string }> = [
    { key: 'req', label: t('modelLeaderboard.summary.requests'), value: formatInt(s.requests), delta: prev ? growthOf(s.requests, prev.requests) : undefined, deltaLabel, hint },
    { key: 'tok', label: t('modelLeaderboard.summary.tokens'), value: formatCompactNumber(s.total_tokens), delta: prev ? growthOf(s.total_tokens, prev.total_tokens) : undefined, deltaLabel, hint },
    { key: 'usr', label: t('modelLeaderboard.summary.users'), value: formatInt(s.users), delta: prev ? growthOf(s.users, prev.users) : undefined, deltaLabel, hint },
    { key: 'mdl', label: t('modelLeaderboard.summary.models'), value: formatInt(s.models), delta: prev ? growthOf(s.models, prev.models) : undefined, deltaLabel, hint }
  ]
  if (data.value.cost_visible) {
    cards.push(
      { key: 'act', label: t('modelLeaderboard.summary.actualCost'), value: `$${formatMoney(s.actual_cost)}`, delta: prev ? growthOf(s.actual_cost ?? 0, prev.actual_cost) : undefined, deltaLabel, hint },
      { key: 'std', label: t('modelLeaderboard.summary.cost'), value: `$${formatMoney(s.cost)}`, delta: prev ? growthOf(s.cost ?? 0, prev.cost) : undefined, deltaLabel, hint }
    )
  }
  return cards
})

// ---------- 颜色 ----------
const palette = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#06b6d4', '#8b5cf6', '#ec4899', '#84cc16', '#f97316', '#14b8a6', '#3b82f6', '#a855f7']
const colorMap = computed(() => {
  const m = new Map<string, string>()
  const order = [...(data.value?.all_time.ranking ?? []), ...(data.value?.monthly.ranking ?? [])]
  for (const r of order) if (!m.has(r.model)) m.set(r.model, palette[m.size % palette.length])
  m.set(LEADERBOARD_OTHER, '#9ca3af')
  return m
})
function colorOf(model: string) {
  return colorMap.value.get(model) ?? '#9ca3af'
}
function labelOf(model: string) {
  return model === LEADERBOARD_OTHER ? t('modelLeaderboard.charts.other') : model
}
function podiumColor(rank: number) {
  return rank === 1 ? 'text-amber-400' : rank === 2 ? 'text-gray-400' : 'text-orange-400'
}
function rankBadge(rank: number) {
  if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
  if (rank === 2) return 'bg-gray-200 text-gray-700 dark:bg-dark-600 dark:text-dark-200'
  if (rank === 3) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300'
  return 'text-gray-500 dark:text-dark-400'
}

const isDark = () => document.documentElement.classList.contains('dark')
const axisColor = () => (isDark() ? '#9ca3af' : '#6b7280')
const gridColor = () => (isDark() ? 'rgba(75,85,99,0.35)' : 'rgba(229,231,235,0.8)')

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
    colors.push('#9ca3af')
  }
  return { labels, datasets: [{ data: values, backgroundColor: colors, borderWidth: 0 }] }
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  cutout: '62%',
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 10, font: { size: 11 } } },
    tooltip: {
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
      borderRadius: 4,
      maxBarThickness: 22
    }]
  }
})

const hBarOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: { callbacks: { label: (ctx: any) => formatInt(ctx.raw) } }
  },
  scales: {
    x: { ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() } },
    y: { ticks: { color: axisColor(), font: { size: 11 } }, grid: { display: false } }
  }
}))

const trendChart = computed(() => {
  const s = period.value?.trend
  if (!s) return { labels: [], datasets: [] }
  const labels = activeTab.value === 'monthly' ? s.labels.map((l) => l.slice(5)) : s.labels
  return {
    labels,
    datasets: s.datasets.map((d) => ({
      label: labelOf(d.model),
      data: trendMetric.value === 'tokens' ? d.tokens : d.requests,
      backgroundColor: colorOf(d.model),
      stack: 'm',
      borderRadius: 2,
      maxBarThickness: 36
    }))
  }
})

const stackedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 10, font: { size: 11 } } },
    tooltip: {
      itemSort: (a: any, b: any) => b.raw - a.raw,
      filter: (item: any) => item.raw > 0,
      callbacks: { label: (ctx: any) => `${ctx.dataset.label}: ${formatInt(ctx.raw)}` }
    }
  },
  scales: {
    x: { stacked: true, ticks: { color: axisColor(), maxRotation: 0, autoSkip: true }, grid: { display: false } },
    y: { stacked: true, ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() } }
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
    maxBarThickness: 18
  })
  return {
    labels: rows.map((r) => r.model),
    datasets: [
      mk('input_tokens', t('modelLeaderboard.charts.input'), '#3b82f6'),
      mk('output_tokens', t('modelLeaderboard.charts.output'), '#10b981'),
      mk('cache_creation_tokens', t('modelLeaderboard.charts.cacheCreate'), '#f59e0b'),
      mk('cache_read_tokens', t('modelLeaderboard.charts.cacheRead'), '#06b6d4')
    ]
  }
})

const tokenMixOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 10, font: { size: 11 } } },
    tooltip: {
      callbacks: {
        label: (ctx: any) => `${ctx.dataset.label}: ${ctx.raw.toFixed(1)}% (${formatCompactNumber(ctx.dataset.raw[ctx.dataIndex])})`
      }
    }
  },
  scales: {
    x: { stacked: true, max: 100, ticks: { color: axisColor(), callback: (v: any) => `${v}%` }, grid: { color: gridColor() } },
    y: { stacked: true, ticks: { color: axisColor(), font: { size: 10 } }, grid: { display: false } }
  }
}))

// 月度明细：调用量柱 + 活跃用户（右轴）
const monthTotalsChart = computed(() => {
  const rows = [...(data.value?.month_rows ?? [])].reverse()
  return {
    labels: rows.map((r) => r.month),
    datasets: [
      { type: 'bar' as const, label: t('modelLeaderboard.metric.requests'), data: rows.map((r) => r.requests), backgroundColor: '#6366f1', borderRadius: 4, yAxisID: 'y', maxBarThickness: 48 },
      { type: 'bar' as const, label: t('modelLeaderboard.metric.users'), data: rows.map((r) => r.users), backgroundColor: '#10b981', borderRadius: 4, yAxisID: 'y1', maxBarThickness: 48 }
    ]
  }
})

const monthTotalsOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: { legend: { position: 'bottom' as const, labels: { color: axisColor(), boxWidth: 10 } } },
  scales: {
    x: { ticks: { color: axisColor() }, grid: { display: false } },
    y: { position: 'left' as const, ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() } },
    y1: { position: 'right' as const, ticks: { color: axisColor(), precision: 0 }, grid: { display: false } }
  }
}))

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
        parts.push(h('span', { class: 'rounded bg-primary-100 px-1.5 py-0.5 text-[10px] font-semibold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300' }, t('modelLeaderboard.table.newEntry')))
      } else {
        const diff = it.prev_rank - it.rank
        const cls = diff > 0 ? 'text-emerald-600 dark:text-emerald-400' : diff < 0 ? 'text-rose-600 dark:text-rose-400' : 'text-gray-400 dark:text-dark-500'
        parts.push(h('span', { class: `text-xs font-medium ${cls}`, title: `#${it.prev_rank} → #${it.rank}` }, diff > 0 ? `▲${diff}` : diff < 0 ? `▼${-diff}` : '—'))
        if (props.showGrowth && it.requests_growth !== null && it.requests_growth !== undefined) {
          parts.push(h('span', { class: `ml-1.5 text-xs ${deltaClass(it.requests_growth)}` }, formatGrowth(it.requests_growth)))
        }
      }
      return h('span', { class: 'inline-flex items-center whitespace-nowrap' }, parts)
    }
  }
})
</script>
