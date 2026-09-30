<template>
  <AppLayout>
    <div class="space-y-6">
      <!-- 筛选栏 -->
      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-x-5 gap-y-3">
          <div :class="segWrapClass" role="tablist">
            <button
              v-for="tab in tabs"
              :key="tab.value"
              type="button"
              role="tab"
              :aria-selected="activeTab === tab.value"
              :class="[segBtnClass, activeTab === tab.value ? segActiveClass : segIdleClass]"
              @click="activeTab = tab.value"
            >
              {{ tab.label }}
            </button>
          </div>

          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('modelLeaderboard.metric.label') }}:</span>
            <div :class="segWrapClass">
              <button
                v-for="m in metrics"
                :key="m"
                type="button"
                :class="[segBtnClass, metric === m ? segActiveClass : segIdleClass]"
                @click="setMetric(m)"
              >
                {{ t(`modelLeaderboard.metric.${m}`) }}
              </button>
            </div>
          </div>

          <div class="flex items-center gap-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('modelLeaderboard.source.label') }}:</span>
            <div :class="segWrapClass">
              <button
                v-for="s in sources"
                :key="s"
                type="button"
                :class="[segBtnClass, source === s ? segActiveClass : segIdleClass]"
                @click="setSource(s)"
              >
                {{ t(`modelLeaderboard.source.${s}`) }}
              </button>
            </div>
          </div>

          <div class="flex items-center gap-2" :title="t('modelLeaderboard.vendor.hint')">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('modelLeaderboard.vendor.label') }}:</span>
            <div class="w-36">
              <Select v-model="vendor" :options="vendorOptions" :aria-label="t('modelLeaderboard.vendor.label')" @change="load()" />
            </div>
          </div>

          <div v-if="activeTab === 'monthly' && monthOptions.length" class="flex items-center gap-2">
            <span class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('modelLeaderboard.month') }}:</span>
            <div class="w-32">
              <Select v-model="month" :options="monthOptions" :aria-label="t('modelLeaderboard.month')" @change="load()" />
            </div>
          </div>

          <div class="ml-auto flex items-center gap-3">
            <span v-if="periodCaption" class="text-xs text-gray-500 dark:text-gray-400">{{ periodCaption }}</span>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load()">
              <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
              {{ t('modelLeaderboard.refresh') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="loading && !data" class="flex items-center justify-center py-16"><LoadingSpinner /></div>
      <div v-else-if="loadFailed && !data" class="card p-8 text-center text-sm text-red-600 dark:text-red-400">
        {{ t('modelLeaderboard.loadFailed') }}
      </div>

      <template v-else-if="data">
        <!-- ============ 当月 / 历史累计 ============ -->
        <template v-if="activeTab !== 'months'">
          <div class="grid grid-cols-2 gap-4 lg:grid-cols-4" :class="data.cost_visible ? 'xl:grid-cols-6' : ''">
            <div v-for="card in summaryCards" :key="card.key" class="card p-4">
              <div class="flex items-center gap-3">
                <div class="shrink-0 rounded-lg p-2" :class="card.iconBg">
                  <Icon :name="card.icon" size="md" :class="card.iconColor" :stroke-width="2" />
                </div>
                <div class="min-w-0">
                  <p class="truncate text-xs font-medium text-gray-500 dark:text-gray-400">{{ card.label }}</p>
                  <p class="truncate text-xl font-bold tabular-nums text-gray-900 dark:text-white" :title="card.title">{{ card.value }}</p>
                  <p class="truncate text-xs text-gray-500 dark:text-gray-400">
                    <template v-if="card.delta !== undefined">
                      <span class="tabular-nums" :class="deltaClass(card.delta)">{{ formatGrowth(card.delta) }}</span>
                      {{ card.deltaLabel }}
                    </template>
                    <template v-else>{{ card.hint || '&nbsp;' }}</template>
                  </p>
                </div>
              </div>
            </div>
          </div>

          <div v-if="!ranking.length" class="card p-10 text-center text-sm text-gray-500 dark:text-dark-400">
            {{ t('modelLeaderboard.empty') }}
          </div>

          <template v-else>
            <!-- 前三名 -->
            <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div v-for="item in top3" :key="item.model" class="card p-4">
                <div class="flex items-center gap-2.5">
                  <span class="flex h-6 w-6 shrink-0 items-center justify-center rounded-full text-xs font-semibold" :class="rankBadgeClass(item.rank)">
                    {{ item.rank }}
                  </span>
                  <ModelIcon :model="item.model" size="18px" />
                  <p class="min-w-0 flex-1 truncate text-sm font-semibold text-gray-900 dark:text-white" :title="item.model">{{ item.model }}</p>
                  <RankMove :item="item" />
                </div>
                <div class="mt-3 flex items-baseline justify-between gap-2">
                  <p class="text-xl font-bold tabular-nums text-gray-900 dark:text-white" :title="formatInt(metricOf(item))">
                    {{ formatMetric(metricOf(item)) }}
                    <span class="text-xs font-normal text-gray-500 dark:text-gray-400">{{ metricLabel }}</span>
                  </p>
                  <span class="text-sm tabular-nums text-gray-500 dark:text-gray-400">{{ pct(shareOf(item)) }}</span>
                </div>
                <div class="mt-2 h-1.5 w-full overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                  <div class="h-full rounded-full bg-primary-500" :style="{ width: pct(shareOf(item)) }" />
                </div>
                <p class="mt-2 truncate text-xs text-gray-500 dark:text-gray-400">
                  {{ tieLabel }} {{ metric === 'tokens' ? formatInt(item.requests) : formatCompactNumber(item.total_tokens) }}
                  · {{ t('modelLeaderboard.table.users') }} {{ formatInt(item.users) }}
                </p>
              </div>
            </div>

            <!-- 图表 -->
            <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
              <div class="card p-4">
                <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.share', { metric: metricLabel }) }}</h3>
                <div class="flex flex-col items-center gap-4 sm:flex-row sm:gap-6">
                  <div class="h-48 w-48 shrink-0">
                    <Doughnut :data="shareChart" :options="doughnutOptions" />
                  </div>
                  <div class="max-h-48 w-full min-w-0 flex-1 overflow-auto">
                    <table class="w-full text-xs">
                      <tbody>
                        <tr v-for="seg in shareLegend" :key="seg.label" class="border-t border-gray-100 first:border-t-0 dark:border-dark-700">
                          <td class="py-1.5 pr-2">
                            <div class="flex min-w-0 items-center gap-2">
                              <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ background: seg.color }" />
                              <span class="truncate text-gray-700 dark:text-gray-300" :title="seg.label">{{ seg.label }}</span>
                            </div>
                          </td>
                          <td class="py-1.5 text-right tabular-nums text-gray-600 dark:text-gray-400">{{ formatMetric(seg.value) }}</td>
                          <td class="w-14 py-1.5 text-right tabular-nums text-gray-900 dark:text-white">{{ pct(seg.share) }}</td>
                        </tr>
                      </tbody>
                    </table>
                  </div>
                </div>
              </div>
              <div class="card p-4">
                <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.topBar', { metric: metricLabel }) }}</h3>
                <div class="h-64">
                  <Bar :data="topBarChart" :options="hBarOptions" />
                </div>
              </div>
            </div>

            <div class="grid grid-cols-1 gap-6 xl:grid-cols-3">
              <div class="card p-4 xl:col-span-2">
                <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t(activeTab === 'monthly' ? 'modelLeaderboard.charts.trendDaily' : 'modelLeaderboard.charts.trendMonthly', { metric: metricLabel }) }}
                </h3>
                <div class="h-72">
                  <Bar :data="trendChart" :options="stackedOptions" />
                </div>
              </div>
              <div class="card p-4">
                <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.tokenMix') }}</h3>
                <div class="h-72">
                  <Bar :data="tokenMixChart" :options="tokenMixOptions" />
                </div>
              </div>
            </div>

            <!-- 完整排行 -->
            <div class="card overflow-hidden">
              <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t('modelLeaderboard.table.title') }}
                  <span class="ml-1 text-xs font-normal text-gray-400">({{ filteredRanking.length }})</span>
                </h3>
                <div class="w-56">
                  <SearchInput v-model="keyword" :placeholder="t('modelLeaderboard.table.search')" :debounce-ms="0" />
                </div>
              </div>
              <div class="overflow-x-auto">
                <table class="w-full min-w-[1000px] text-sm">
                  <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800/50 dark:text-gray-400">
                    <tr>
                      <th
                        v-for="col in columns"
                        :key="col.key"
                        class="cursor-pointer select-none whitespace-nowrap px-3 py-2.5 font-medium hover:text-gray-900 dark:hover:text-white"
                        :class="[col.left ? 'text-left' : 'text-right', col.key === metricColumn ? 'text-gray-900 dark:text-white' : '']"
                        @click="toggleSort(col.key)"
                      >
                        {{ col.label }}
                        <span v-if="sortKey === col.key" class="text-primary-500">{{ sortDesc ? '↓' : '↑' }}</span>
                      </th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                    <tr v-for="row in filteredRanking" :key="row.model" class="transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/30">
                      <td class="px-3 py-2.5 text-left">
                        <span class="inline-flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold tabular-nums" :class="rankBadgeClass(row.rank)">
                          {{ row.rank }}
                        </span>
                      </td>
                      <td class="max-w-[260px] px-3 py-2.5 text-left">
                        <div class="flex items-center gap-2">
                          <ModelIcon :model="row.model" size="16px" />
                          <span class="truncate font-medium text-gray-900 dark:text-white" :title="row.model">{{ row.model }}</span>
                        </div>
                      </td>
                      <td class="px-3 py-2.5 text-right tabular-nums" :class="cellClass('requests')">{{ formatInt(row.requests) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums" :class="cellClass('total_tokens')" :title="formatInt(row.total_tokens)">{{ formatCompactNumber(row.total_tokens) }}</td>
                      <td class="px-3 py-2.5 text-right">
                        <div class="flex items-center justify-end gap-2">
                          <div class="h-1.5 w-20 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
                            <div class="h-full rounded-full bg-primary-500" :style="{ width: pct(shareOf(row)) }" />
                          </div>
                          <span class="w-12 tabular-nums text-gray-600 dark:text-gray-400">{{ pct(shareOf(row)) }}</span>
                        </div>
                      </td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ formatCompactNumber(row.input_tokens) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ formatCompactNumber(row.output_tokens) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ pct(row.cache_hit_rate) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-400">{{ formatInt(row.users) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ formatMs(row.avg_duration_ms) }}</td>
                      <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ formatMs(row.avg_first_token_ms) }}</td>
                      <td class="px-3 py-2.5 text-right"><RankMove :item="row" show-growth /></td>
                      <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-green-600 dark:text-green-400">${{ formatMoney(row.actual_cost) }}</td>
                      <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-gray-400 dark:text-gray-500">${{ formatMoney(row.cost) }}</td>
                      <td class="whitespace-nowrap px-3 py-2.5 text-right text-xs text-gray-500 dark:text-gray-400">{{ shortTime(row.last_used_at) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </template>
        </template>

        <!-- ============ 月度明细 ============ -->
        <div v-else-if="!data.month_rows.length" class="card p-10 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('modelLeaderboard.empty') }}
        </div>
        <template v-else>
          <div class="card p-4">
            <h3 class="mb-4 text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.charts.trendMonthly', { metric: metricLabel }) }}</h3>
            <div class="h-72">
              <Bar :data="monthTotalsChart" :options="monthTotalsOptions" />
            </div>
          </div>
          <div class="card overflow-hidden">
            <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
              <h3 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('modelLeaderboard.months.title') }}</h3>
            </div>
            <div class="overflow-x-auto">
              <table class="w-full min-w-[820px] text-sm">
                <thead class="bg-gray-50 text-xs text-gray-500 dark:bg-dark-800/50 dark:text-gray-400">
                  <tr>
                    <th class="px-3 py-2.5 text-left font-medium">{{ t('modelLeaderboard.months.month') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium" :class="metric === 'requests' ? 'text-gray-900 dark:text-white' : ''">{{ t('modelLeaderboard.table.requests') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium" :class="metric === 'tokens' ? 'text-gray-900 dark:text-white' : ''">{{ t('modelLeaderboard.table.tokens') }}</th>
                    <th class="px-3 py-2.5 text-right font-medium">{{ t('modelLeaderboard.table.growth') }}</th>
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
                    class="cursor-pointer transition-colors hover:bg-gray-50 dark:hover:bg-dark-800/30"
                    @click="openMonth(row.month)"
                  >
                    <td class="whitespace-nowrap px-3 py-2.5 font-medium text-primary-600 hover:underline dark:text-primary-400">{{ row.month }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums" :class="cellClass('requests')">{{ formatInt(row.requests) }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums" :class="cellClass('total_tokens')" :title="formatInt(row.total_tokens)">{{ formatCompactNumber(row.total_tokens) }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums" :class="deltaClass(monthGrowth(i))">{{ formatGrowth(monthGrowth(i)) }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-400">{{ formatInt(row.users) }}</td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-600 dark:text-gray-400">{{ formatInt(row.models) }}</td>
                    <td class="max-w-[220px] px-3 py-2.5">
                      <div class="flex items-center gap-2">
                        <ModelIcon v-if="row.top_model" :model="row.top_model" size="16px" />
                        <span class="truncate text-gray-900 dark:text-white" :title="row.top_model">{{ row.top_model || '-' }}</span>
                      </div>
                    </td>
                    <td class="px-3 py-2.5 text-right tabular-nums text-gray-500 dark:text-gray-400">{{ pct(row.top_model_share) }}</td>
                    <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-green-600 dark:text-green-400">${{ formatMoney(row.actual_cost) }}</td>
                    <td v-if="data.cost_visible" class="px-3 py-2.5 text-right tabular-nums text-gray-400 dark:text-gray-500">${{ formatMoney(row.cost) }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </template>

        <p class="text-xs text-gray-400 dark:text-dark-500">
          {{ t('modelLeaderboard.footnote', { tz: data.timezone, metric: metricLabel, tie: tieLabel }) }} · {{ shortTime(data.generated_at) }}
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
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { formatCompactNumber } from '@/utils/format'
import {
  getModelLeaderboard,
  LEADERBOARD_OTHER,
  type LeaderboardMetric,
  type LeaderboardRankItem,
  type LeaderboardSource,
  type ModelLeaderboardResponse
} from '@/api/modelLeaderboard'

ChartJS.register(ArcElement, BarElement, CategoryScale, LinearScale, Tooltip, Legend)

type Tab = 'monthly' | 'allTime' | 'months'
type IconName = InstanceType<typeof Icon>['$props']['name']

const { t, locale } = useI18n()
const route = useRoute()
const router = useRouter()

const segWrapClass = 'inline-flex rounded-lg border border-gray-200 bg-gray-50 p-0.5 dark:border-dark-700 dark:bg-dark-800'
const segBtnClass = 'rounded-md px-3 py-1 text-sm font-medium transition-colors'
const segActiveClass = 'bg-white text-gray-900 shadow-sm dark:bg-dark-700 dark:text-white'
const segIdleClass = 'text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200'

const sources: LeaderboardSource[] = ['requested', 'upstream']
const metrics: LeaderboardMetric[] = ['requests', 'tokens']

const initialTab = (['monthly', 'allTime', 'months'] as Tab[]).includes(route.query.tab as Tab)
  ? (route.query.tab as Tab)
  : 'monthly'
const activeTab = ref<Tab>(initialTab)
const source = ref<LeaderboardSource>(route.query.source === 'upstream' ? 'upstream' : 'requested')
const metric = ref<LeaderboardMetric>(route.query.metric === 'tokens' ? 'tokens' : 'requests')
const month = ref<string>(typeof route.query.month === 'string' ? route.query.month : '')
// 与后端 LeaderboardVendorOf 的分类键一致；品牌名不翻译
const vendorNames: Record<string, string> = {
  claude: 'Claude',
  openai: 'OpenAI',
  gemini: 'Gemini',
  deepseek: 'DeepSeek',
  moonshot: 'Moonshot',
  zhipu: 'Zhipu GLM',
  qwen: 'Qwen',
  minimax: 'MiniMax',
  xai: 'xAI',
  doubao: 'Doubao',
  mistral: 'Mistral',
  meta: 'Meta',
  other: ''
}
const queryVendor = typeof route.query.vendor === 'string' ? route.query.vendor : ''
const vendor = ref<string>(Object.prototype.hasOwnProperty.call(vendorNames, queryVendor) ? queryVendor : '')
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

async function load() {
  controller?.abort()
  controller = new AbortController()
  loading.value = true
  loadFailed.value = false
  try {
    const resp = await getModelLeaderboard(
      { source: source.value, month: month.value || undefined, metric: metric.value, vendor: vendor.value || undefined },
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

function setMetric(m: LeaderboardMetric) {
  if (metric.value === m) return
  metric.value = m
  sortKey.value = 'rank'
  sortDesc.value = false
  load()
}

function openMonth(m: string) {
  month.value = m
  activeTab.value = 'monthly'
  load()
}

watch([activeTab, source, metric, month, vendor], () => {
  const query: Record<string, string> = { ...(route.query as Record<string, string>) }
  query.tab = activeTab.value
  query.source = source.value
  query.metric = metric.value
  if (month.value) query.month = month.value
  if (vendor.value) query.vendor = vendor.value
  else delete query.vendor
  router.replace({ query }).catch(() => {})
})

onMounted(() => load())

const monthOptions = computed(() => (data.value?.months ?? []).map((m) => ({ value: m, label: m })))

function vendorLabel(v: string) {
  return v === 'other' ? t('modelLeaderboard.vendor.other') : vendorNames[v] || v
}
// 仅列出有数据的供应商；当前选中项即使切换口径后无数据也保留，避免下拉框显示空白
const vendorOptions = computed(() => {
  const list = [...(data.value?.vendors ?? [])]
  if (vendor.value && !list.includes(vendor.value)) list.push(vendor.value)
  return [
    { value: '', label: t('modelLeaderboard.vendor.all') },
    ...list.map((v) => ({ value: v, label: vendorLabel(v) }))
  ]
})
const period = computed(() => (activeTab.value === 'allTime' ? data.value?.all_time : data.value?.monthly))
const ranking = computed(() => period.value?.ranking ?? [])
const top3 = computed(() => ranking.value.slice(0, 3))

// ---------- 统计口径 ----------
const isTokens = computed(() => metric.value === 'tokens')
const metricLabel = computed(() => t(`modelLeaderboard.metric.${metric.value}`))
const tieLabel = computed(() => t(`modelLeaderboard.metric.${isTokens.value ? 'requests' : 'tokens'}`))
const metricColumn = computed(() => (isTokens.value ? 'total_tokens' : 'requests'))

function metricOf(r: { requests: number; total_tokens: number }) {
  return isTokens.value ? r.total_tokens : r.requests
}
function shareOf(r: LeaderboardRankItem) {
  return isTokens.value ? r.token_share : r.request_share
}
function growthOfItem(r: LeaderboardRankItem) {
  return isTokens.value ? r.tokens_growth : r.requests_growth
}
function formatMetric(v: number) {
  return isTokens.value ? formatCompactNumber(v) : formatInt(v)
}
function cellClass(col: string) {
  return col === metricColumn.value
    ? 'font-semibold text-gray-900 dark:text-white'
    : 'text-gray-600 dark:text-gray-400'
}

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
  if (g === null || g === undefined || g === 0) return 'text-gray-400 dark:text-gray-500'
  return g > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-600 dark:text-red-400'
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
function rankBadgeClass(rank: number) {
  if (rank === 1) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
  if (rank === 2) return 'bg-gray-200 text-gray-700 dark:bg-dark-600 dark:text-gray-200'
  if (rank === 3) return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
  return 'text-gray-500 dark:text-gray-400'
}

const periodCaption = computed(() => {
  const p = activeTab.value === 'months' ? data.value?.all_time : period.value
  if (!p) return ''
  const start = (p.summary.first_at || p.start || '').slice(0, 10)
  const end = (activeTab.value === 'monthly' ? p.end : p.summary.last_at || p.end || '').slice(0, 10)
  return start && end ? t('modelLeaderboard.summary.range', { start, end }) : ''
})

// ---------- 汇总卡片 ----------
type SummaryCard = {
  key: string
  label: string
  value: string
  title?: string
  icon: IconName
  iconBg: string
  iconColor: string
  delta?: number | null
  deltaLabel?: string
  hint?: string
}

const summaryCards = computed<SummaryCard[]>(() => {
  const p = period.value
  if (!p || !data.value) return []
  const s = p.summary
  const prev = activeTab.value === 'monthly' ? p.prev : undefined
  const deltaLabel = data.value.is_current ? t('modelLeaderboard.summary.vsPrev') : t('modelLeaderboard.summary.vsPrevMonth')
  const hint = activeTab.value === 'allTime' && s.first_at
    ? t('modelLeaderboard.summary.since', { date: s.first_at.slice(0, 10) })
    : undefined
  const delta = (cur: number, before: number | undefined) => (prev ? growthOf(cur, before) : undefined)
  const cards: SummaryCard[] = [
    {
      key: 'req', label: t('modelLeaderboard.summary.requests'), value: formatInt(s.requests),
      icon: 'chart', iconBg: 'bg-blue-100 dark:bg-blue-900/30', iconColor: 'text-blue-600 dark:text-blue-400',
      delta: delta(s.requests, prev?.requests), deltaLabel, hint
    },
    {
      key: 'tok', label: t('modelLeaderboard.summary.tokens'), value: formatCompactNumber(s.total_tokens), title: formatInt(s.total_tokens),
      icon: 'database', iconBg: 'bg-amber-100 dark:bg-amber-900/30', iconColor: 'text-amber-600 dark:text-amber-400',
      delta: delta(s.total_tokens, prev?.total_tokens), deltaLabel, hint
    },
    {
      key: 'usr', label: t('modelLeaderboard.summary.users'), value: formatInt(s.users),
      icon: 'users', iconBg: 'bg-emerald-100 dark:bg-emerald-900/30', iconColor: 'text-emerald-600 dark:text-emerald-400',
      delta: delta(s.users, prev?.users), deltaLabel, hint
    },
    {
      key: 'mdl', label: t('modelLeaderboard.summary.models'), value: formatInt(s.models),
      icon: 'cube', iconBg: 'bg-purple-100 dark:bg-purple-900/30', iconColor: 'text-purple-600 dark:text-purple-400',
      delta: delta(s.models, prev?.models), deltaLabel, hint
    }
  ]
  if (data.value.cost_visible) {
    cards.push(
      {
        key: 'act', label: t('modelLeaderboard.summary.actualCost'), value: `$${formatMoney(s.actual_cost)}`,
        icon: 'dollar', iconBg: 'bg-green-100 dark:bg-green-900/30', iconColor: 'text-green-600 dark:text-green-400',
        delta: delta(s.actual_cost ?? 0, prev?.actual_cost), deltaLabel, hint
      },
      {
        key: 'std', label: t('modelLeaderboard.summary.cost'), value: `$${formatMoney(s.cost)}`,
        icon: 'creditCard', iconBg: 'bg-gray-100 dark:bg-dark-700', iconColor: 'text-gray-600 dark:text-gray-300',
        delta: delta(s.cost ?? 0, prev?.cost), deltaLabel, hint
      }
    )
  }
  return cards
})

// ---------- 颜色 ----------
const palette = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1', '#84cc16', '#06b6d4', '#a855f7']
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
function labelOf(model: string) {
  return model === LEADERBOARD_OTHER ? t('modelLeaderboard.charts.other') : model
}

const isDark = () => document.documentElement.classList.contains('dark')
const axisColor = () => (isDark() ? '#9ca3af' : '#6b7280')
const gridColor = () => (isDark() ? 'rgba(75,85,99,0.35)' : 'rgba(229,231,235,0.8)')
const legendBottom = () => ({
  position: 'bottom' as const,
  labels: { color: axisColor(), boxWidth: 10, boxHeight: 10, font: { size: 11 } }
})

// ---------- 图表 ----------
const shareChart = computed(() => {
  const rows = ranking.value
  const top = rows.slice(0, 8)
  const rest = rows.slice(8).reduce((a, r) => a + metricOf(r), 0)
  const labels = top.map((r) => r.model)
  const values = top.map((r) => metricOf(r))
  const colors = top.map((r) => colorOf(r.model))
  if (rest > 0) {
    labels.push(t('modelLeaderboard.charts.other'))
    values.push(rest)
    colors.push(otherColor)
  }
  return { labels, datasets: [{ data: values, backgroundColor: colors, borderWidth: 0 }] }
})

const shareLegend = computed(() => {
  const c = shareChart.value
  const values = c.datasets[0].data
  const total = values.reduce((a, b) => a + b, 0)
  return c.labels.map((label, i) => ({
    label,
    color: c.datasets[0].backgroundColor[i],
    value: values[i],
    share: total ? values[i] / total : 0
  }))
})

const doughnutOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
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

const topBarChart = computed(() => {
  const rows = ranking.value.slice(0, 10)
  return {
    labels: rows.map((r) => r.model),
    datasets: [{
      label: metricLabel.value,
      data: rows.map((r) => metricOf(r)),
      backgroundColor: rows.map((r) => colorOf(r.model)),
      borderRadius: 4,
      maxBarThickness: 18
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
  const raw = s.labels ?? []
  const labels = activeTab.value === 'monthly' ? raw.map((l) => l.slice(5)) : raw
  return {
    labels,
    datasets: (s.datasets ?? []).map((d) => ({
      label: labelOf(d.model),
      data: isTokens.value ? d.tokens : d.requests,
      backgroundColor: colorOf(d.model),
      stack: 'm',
      maxBarThickness: 36
    }))
  }
})

const stackedOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { mode: 'index' as const, intersect: false },
  plugins: {
    legend: legendBottom(),
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
  const rows = [...ranking.value].sort((a, b) => b.total_tokens - a.total_tokens).slice(0, 10)
  const mk = (key: keyof LeaderboardRankItem, label: string, color: string) => ({
    label,
    data: rows.map((r) => ((r[key] as number) / (r.total_tokens || 1)) * 100),
    raw: rows.map((r) => r[key] as number),
    backgroundColor: color,
    stack: 't',
    maxBarThickness: 16
  })
  return {
    labels: rows.map((r) => r.model),
    datasets: [
      mk('input_tokens', t('modelLeaderboard.charts.input'), '#3b82f6'),
      mk('output_tokens', t('modelLeaderboard.charts.output'), '#10b981'),
      mk('cache_creation_tokens', t('modelLeaderboard.charts.cacheCreate'), '#f59e0b'),
      mk('cache_read_tokens', t('modelLeaderboard.charts.cacheRead'), '#8b5cf6')
    ]
  }
})

const tokenMixOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: legendBottom(),
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

// 月度明细：当前口径柱 + 活跃用户（右轴）
const monthTotalsChart = computed(() => {
  const rows = [...(data.value?.month_rows ?? [])].reverse()
  return {
    labels: rows.map((r) => r.month),
    datasets: [
      { label: metricLabel.value, data: rows.map((r) => metricOf(r)), backgroundColor: '#3b82f6', borderRadius: 4, yAxisID: 'y', maxBarThickness: 40 },
      { label: t('modelLeaderboard.metric.users'), data: rows.map((r) => r.users), backgroundColor: '#10b981', borderRadius: 4, yAxisID: 'y1', maxBarThickness: 40 }
    ]
  }
})

const monthTotalsOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: legendBottom(),
    tooltip: { callbacks: { label: (ctx: any) => `${ctx.dataset.label}: ${formatInt(ctx.raw)}` } }
  },
  scales: {
    x: { ticks: { color: axisColor() }, grid: { display: false } },
    y: { position: 'left' as const, ticks: { color: axisColor(), callback: (v: any) => formatCompactNumber(Number(v)) }, grid: { color: gridColor() } },
    y1: { position: 'right' as const, ticks: { color: axisColor(), precision: 0 }, grid: { display: false } }
  }
}))

function monthGrowth(i: number): number | null {
  const rows = data.value?.month_rows ?? []
  const prev = rows[i + 1]
  return prev ? growthOf(metricOf(rows[i]), metricOf(prev)) : null
}

// ---------- 表格 ----------
const columns = computed(() => {
  const cols: Array<{ key: string; label: string; left?: boolean }> = [
    { key: 'rank', label: t('modelLeaderboard.table.rank'), left: true },
    { key: 'model', label: t('modelLeaderboard.table.model'), left: true },
    { key: 'requests', label: t('modelLeaderboard.table.requests') },
    { key: 'total_tokens', label: t('modelLeaderboard.table.tokens') },
    { key: 'share', label: t('modelLeaderboard.table.share') },
    { key: 'input_tokens', label: t('modelLeaderboard.table.input') },
    { key: 'output_tokens', label: t('modelLeaderboard.table.output') },
    { key: 'cache_hit_rate', label: t('modelLeaderboard.table.cacheHit') },
    { key: 'users', label: t('modelLeaderboard.table.users') },
    { key: 'avg_duration_ms', label: t('modelLeaderboard.table.latency') },
    { key: 'avg_first_token_ms', label: t('modelLeaderboard.table.ttft') },
    { key: 'growth', label: t('modelLeaderboard.table.growth') }
  ]
  if (data.value?.cost_visible) {
    cols.push(
      { key: 'actual_cost', label: t('modelLeaderboard.table.actualCost') },
      { key: 'cost', label: t('modelLeaderboard.table.cost') }
    )
  }
  cols.push({ key: 'last_used_at', label: t('modelLeaderboard.table.lastUsed') })
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

function sortValue(r: LeaderboardRankItem, key: string) {
  if (key === 'share') return shareOf(r)
  if (key === 'growth') return growthOfItem(r)
  return r[key as keyof LeaderboardRankItem]
}

const filteredRanking = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const rows = ranking.value.filter((r) => !kw || r.model.toLowerCase().includes(kw))
  const key = sortKey.value
  const dir = sortDesc.value ? -1 : 1
  return [...rows].sort((a, b) => {
    const av = sortValue(a, key)
    const bv = sortValue(b, key)
    if (av === bv) return a.rank - b.rank
    if (av === null || av === undefined) return 1
    if (bv === null || bv === undefined) return -1
    if (typeof av === 'string' && typeof bv === 'string') return av.localeCompare(bv) * dir
    return ((av as number) - (bv as number)) * dir
  })
})

// ---------- 排名变化 ----------
const RankMove = defineComponent({
  props: {
    item: { type: Object as () => LeaderboardRankItem, required: true },
    showGrowth: { type: Boolean, default: false }
  },
  setup(props) {
    return () => {
      const it = props.item
      if (it.prev_rank === null || it.prev_rank === undefined) {
        return h('span', { class: 'badge badge-primary whitespace-nowrap' }, t('modelLeaderboard.table.newEntry'))
      }
      const diff = it.prev_rank - it.rank
      const cls = diff > 0 ? 'text-emerald-600 dark:text-emerald-400' : diff < 0 ? 'text-red-600 dark:text-red-400' : 'text-gray-400 dark:text-gray-500'
      const parts = [h('span', { class: `text-xs font-medium ${cls}`, title: `#${it.prev_rank} → #${it.rank}` }, diff > 0 ? `↑${diff}` : diff < 0 ? `↓${-diff}` : '—')]
      const g = growthOfItem(it)
      if (props.showGrowth && g !== null && g !== undefined) {
        parts.push(h('span', { class: `ml-1.5 text-xs tabular-nums ${deltaClass(g)}` }, formatGrowth(g)))
      }
      return h('span', { class: 'inline-flex items-center whitespace-nowrap' }, parts)
    }
  }
})
</script>
