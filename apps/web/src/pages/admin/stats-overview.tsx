import * as echarts from 'echarts/core'
import { AlertCircle, MessageSquare, Timer, Users, Zap } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { ApiError } from '@/api/client'
import { Button } from '@/components/ui/button'
import { EChartsWrapper } from '@/components/ui/echarts'
import { Input } from '@/components/ui/input'
import { useQAMetricsQueries } from '@/features/qa-admin/qa-admin.queries'
import type {
  QAIntentDistributionItem,
  QAMetricsOverview,
  QAMetricsTrendPoint,
  QATopQuery,
} from '@/features/qa-admin/qa-admin.types'

// ── Theme helpers ──

function isDark(): boolean {
  if (typeof document === 'undefined') return false
  return document.documentElement.classList.contains('dark')
}

function textColor(): string {
  return isDark() ? '#a1a1aa' : '#71717a'
}

function borderColor(): string {
  return isDark() ? '#27272a' : '#e4e4e7'
}

function cardBg(): string {
  return isDark() ? '#18181b' : '#fff'
}

// ── Color palettes ──

const ROSE_COLORS = ['#ec4899', '#d946ef', '#a855f7', '#6366f1']
const PIE_COLORS = [
  '#0d9488',
  '#0891b2',
  '#0284c7',
  '#4f46e5',
  '#7c3aed',
  '#db2777',
  '#ea580c',
  '#65a30d',
]

// ── Helpers ──

function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.requestId ? `${error.message}（requestId: ${error.requestId}）` : error.message
  }
  return error instanceof Error ? error.message : '未知错误'
}

function formatDate(value: string | undefined): string {
  return value ? new Date(value).toLocaleDateString() : '-'
}

function formatNumber(value: number | undefined): string {
  return value === undefined ? '-' : value.toLocaleString()
}

// ── Shared UI ──

function SectionState({ message, tone }: { message: string; tone: 'empty' | 'error' }) {
  return (
    <div
      className={
        tone === 'error'
          ? 'flex items-start gap-2 rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive'
          : 'flex items-start gap-2 rounded-lg border border-dashed border-border bg-muted/20 p-4 text-sm text-muted-foreground'
      }
    >
      <AlertCircle aria-hidden="true" className="mt-0.5 size-4 shrink-0" />
      <span>{message}</span>
    </div>
  )
}

function ChartSkeleton({ height }: { height: number }) {
  return <div className="skeleton-shimmer rounded-lg" style={{ height: `${height}px` }} />
}

function SkeletonBlock({ className }: { className?: string }) {
  return <div className={`${className ?? 'h-6 w-20'} rounded skeleton-shimmer`} />
}

// ── Top stat cards ──

function StatCard({
  label,
  value,
  icon: Icon,
  suffix,
  accent,
}: {
  label: string
  value: string
  icon: React.ComponentType<{ className?: string; style?: React.CSSProperties }>
  suffix?: string
  accent: string
}) {
  return (
    <div className="flex items-center gap-4 rounded-lg border border-border bg-card p-4 hover:-translate-y-0.5 hover:shadow-md transition-all duration-200">
      <div
        className="flex size-10 shrink-0 items-center justify-center rounded-lg"
        style={{ backgroundColor: `${accent}18` }}
      >
        <Icon className="size-5" style={{ color: accent }} />
      </div>
      <div className="min-w-0">
        <p className="text-xs text-muted-foreground">{label}</p>
        <p className="text-xl font-bold text-foreground tabular-nums">
          {value}
          {suffix && (
            <span className="ml-0.5 text-sm font-normal text-muted-foreground">{suffix}</span>
          )}
        </p>
      </div>
    </div>
  )
}

function StatCards({ overview }: { overview: QAMetricsOverview }) {
  const v = (k: string) => (overview as Record<string, number | undefined>)[k]
  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <StatCard
        label="今日问答"
        value={formatNumber(v('todayQaCount'))}
        icon={Zap}
        accent="#6366f1"
      />
      <StatCard
        label="今日活跃用户"
        value={formatNumber(v('activeUsersToday'))}
        icon={Users}
        accent="#06b6d4"
      />
      <StatCard
        label="平均延迟"
        value={v('avgLatencyMs') != null ? `${Math.round(v('avgLatencyMs')!)}` : '-'}
        icon={Timer}
        suffix="ms"
        accent="#f59e0b"
      />
      <StatCard
        label="会话数"
        value={formatNumber(v('conversationCount'))}
        icon={MessageSquare}
        accent="#10b981"
      />
    </div>
  )
}

// ── Chart 1: Volume metrics Nightingale Rose ──

const ROSE_METRICS: { key: keyof QAMetricsOverview; label: string }[] = [
  { key: 'totalQaCount', label: '总问答次数' },
  { key: 'totalQuestionCount', label: '问题总数' },
  { key: 'knowledgeBaseCount', label: '知识库数量' },
  { key: 'documentCount', label: '文档总数' },
]

function MetricsRoseChart({ overview }: { overview: QAMetricsOverview }) {
  // Store real values for tooltip lookup
  const realValues = useMemo(
    () => Object.fromEntries(ROSE_METRICS.map(({ key, label }) => [label, overview[key] ?? 0])),
    [overview],
  )

  // All petals use uniform visual weight for a perfect rose shape
  const data = useMemo(
    () =>
      ROSE_METRICS.map(({ label }, i) => ({
        name: label,
        value: 100 + i * 20, // small gradient for slight visual variety
        real: realValues[label],
      })),
    [realValues],
  )

  const option = useMemo(
    () => ({
      tooltip: {
        trigger: 'item' as const,
        backgroundColor: cardBg(),
        borderColor: borderColor(),
        textStyle: { fontSize: 13 },
        formatter: (params: { name: string; data: { real: number } }) =>
          `${params.name}: ${params.data.real.toLocaleString()}`,
      },
      series: [
        {
          name: '容量指标',
          type: 'pie',
          radius: ['30%', '78%'],
          center: ['50%', '50%'],
          roseType: 'area',
          itemStyle: {
            borderRadius: 8,
            borderColor: cardBg(),
            borderWidth: 3,
          },
          color: ROSE_COLORS,
          label: {
            show: true,
            color: textColor(),
            fontSize: 12,
            formatter: (params: { name: string; data: { real: number } }) =>
              `${params.name}\n${params.data.real.toLocaleString()}`,
          },
          emphasis: {
            label: {
              fontSize: 15,
              fontWeight: 'bold',
              formatter: (params: { name: string; data: { real: number } }) =>
                `${params.name}\n${params.data.real.toLocaleString()}`,
            },
            itemStyle: { shadowBlur: 14, shadowColor: 'rgba(0,0,0,0.2)' },
          },
          data,
        },
      ],
    }),
    [data],
  )

  return <EChartsWrapper option={option} style={{ minHeight: 300 }} />
}

// ── Chart 2: Trend Bar + Line + Dotted overlay ──

function TrendChart({ points }: { points: QAMetricsTrendPoint[] }) {
  const normalized = useMemo(
    () => points.map((p) => ({ date: p.date, count: p.count ?? p.questionCount ?? 0 })),
    [points],
  )
  const tc = textColor()
  const bc = borderColor()

  const option = useMemo(() => {
    const categories = normalized.map((p) => p.date)
    const realData = normalized.map((p) => p.count)
    const maxVal = Math.max(1, ...realData)
    // Use real values directly; yAxis.min=0 ensures axis starts at zero
    const barData = realData.map((v) => v + 0.5) // imperceptible offset so zero bars still render
    const lineData = barData.map((v) => v + maxVal * 0.15) // line sits slightly above bars

    return {
      grid: { top: 40, right: 24, bottom: 32, left: 48 },
      xAxis: {
        type: 'category' as const,
        data: categories,
        axisLine: { lineStyle: { color: bc } },
        axisTick: { show: false },
        axisLabel: { color: tc, fontSize: 10, rotate: categories.length > 14 ? 45 : 0 },
      },
      yAxis: {
        type: 'value' as const,
        min: 0,
        axisLine: { show: true, lineStyle: { color: bc } },
        splitLine: { lineStyle: { color: bc, type: 'dashed' as const } },
        axisLabel: { show: false },
        axisTick: { show: false },
      },
      tooltip: {
        trigger: 'axis' as const,
        backgroundColor: cardBg(),
        borderColor: bc,
        textStyle: { fontSize: 13 },
        formatter: (params: unknown) => {
          const items = params as { seriesName: string; dataIndex: number }[] | null
          if (!items || items.length === 0 || !items[0]) return ''
          const idx = items[0].dataIndex
          const real = realData[idx]
          if (real === undefined) return ''
          return `${categories[idx]}<br/>问答数量: ${real.toLocaleString()}`
        },
      },
      animationDuration: 1400,
      animationEasing: 'elasticOut' as const,
      animationDelay: (idx: number) => idx * 40,
      series: [
        {
          name: 'backdrop',
          type: 'bar',
          barGap: '-100%',
          barWidth: '50%',
          itemStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(99,102,241,0.25)' },
              { offset: 0.3, color: 'rgba(99,102,241,0.08)' },
              { offset: 1, color: 'rgba(99,102,241,0)' },
            ]),
            borderRadius: [6, 6, 0, 0],
          },
          z: 1,
          data: lineData,
        },
        {
          name: '问答数量',
          type: 'bar',
          barWidth: '50%',
          itemStyle: {
            borderRadius: [4, 4, 0, 0],
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: '#818cf8' },
              { offset: 1, color: '#4f46e5' },
            ]),
          },
          emphasis: {
            itemStyle: {
              color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
                { offset: 0, color: '#a5b4fc' },
                { offset: 1, color: '#6366f1' },
              ]),
            },
          },
          z: 3,
          data: barData,
        },
        {
          name: 'trend-line',
          type: 'line',
          smooth: true,
          showSymbol: true,
          symbol: 'emptyCircle',
          symbolSize: 10,
          lineStyle: { color: '#a5b4fc', width: 2 },
          itemStyle: { color: '#818cf8', borderColor: '#a5b4fc', borderWidth: 2 },
          z: 4,
          data: lineData,
          label: { show: false },
        },
        {
          name: 'dot-matrix',
          type: 'pictorialBar',
          symbol: 'rect',
          symbolRepeat: true,
          symbolSize: [10, 3],
          symbolMargin: 2,
          itemStyle: { color: isDark() ? 'rgba(24,24,27,0.6)' : 'rgba(255,255,255,0.7)' },
          z: 2,
          data: lineData,
        },
      ],
    }
  }, [normalized, tc, bc])

  return <EChartsWrapper option={option} style={{ minHeight: 320 }} />
}

// ── Chart 3: Intent Pie with gaps + label lines ──

function IntentGapPie({ items }: { items: QAIntentDistributionItem[] }) {
  const tc = textColor()
  const data = useMemo(
    () => items.map((item) => ({ name: item.label ?? item.intent, value: item.count })),
    [items],
  )

  const option = useMemo(
    () => ({
      color: PIE_COLORS,
      tooltip: {
        trigger: 'item' as const,
        backgroundColor: cardBg(),
        borderColor: borderColor(),
        textStyle: { fontSize: 13 },
        formatter: '{b}: {c} ({d}%)',
      },
      series: [
        {
          name: '意图分布',
          type: 'pie',
          radius: ['38%', '70%'],
          center: ['50%', '52%'],
          padAngle: 3,
          itemStyle: {
            borderRadius: 6,
            borderColor: cardBg(),
            borderWidth: 3,
          },
          label: {
            show: true,
            position: 'outside' as const,
            color: tc,
            fontSize: 11,
            formatter: '{b} {d}%',
          },
          labelLine: {
            show: true,
            length: 18,
            length2: 24,
            lineStyle: { color: tc, width: 1 },
          },
          emphasis: {
            scaleSize: 8,
            label: { fontSize: 14, fontWeight: 'bold' },
            itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.15)' },
          },
          data,
        },
      ],
    }),
    [data, tc],
  )

  return <EChartsWrapper option={option} style={{ minHeight: 320 }} />
}

// ── Table ──

function TopQueriesTable({ queries }: { queries: QATopQuery[] }) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <table className="w-full text-left text-sm">
        <thead className="bg-muted/40 text-xs text-muted-foreground">
          <tr>
            <th className="px-3 py-2 font-medium">问题</th>
            <th className="w-24 px-3 py-2 font-medium">次数</th>
            <th className="w-28 px-3 py-2 font-medium">平均延迟</th>
            <th className="w-36 px-3 py-2 font-medium">最近提问</th>
          </tr>
        </thead>
        <tbody>
          {queries.map((query) => (
            <tr
              key={`${query.query}-${query.lastAskedAt ?? ''}`}
              className="border-t border-border transition-colors duration-150 hover:bg-muted/30"
            >
              <td className="break-words px-3 py-2 text-foreground">{query.query}</td>
              <td className="px-3 py-2 font-mono">{query.count}</td>
              <td className="px-3 py-2 font-mono">
                {query.avgLatencyMs === undefined ? '-' : `${query.avgLatencyMs}ms`}
              </td>
              <td className="px-3 py-2 text-xs text-muted-foreground">
                {formatDate(query.lastAskedAt)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// ── Page ──

export function StatsOverviewPage() {
  const [overviewDays, setOverviewDays] = useState('1')
  const [trendDays, setTrendDays] = useState('30')
  const [rankingDays, setRankingDays] = useState('7')
  const [rankingLimit, setRankingLimit] = useState('10')
  const [, setDarkModeKey] = useState(0)

  useEffect(() => {
    const observer = new MutationObserver(() => setDarkModeKey((k) => k + 1))
    observer.observe(document.documentElement, { attributes: true, attributeFilter: ['class'] })
    return () => observer.disconnect()
  }, [])

  const filters = useMemo(
    () => ({
      overviewDays: Math.max(1, Number(overviewDays) || 1),
      trendDays: Math.max(1, Number(trendDays) || 30),
      rankingDays: Math.max(1, Number(rankingDays) || 7),
      rankingLimit: Math.max(1, Number(rankingLimit) || 10),
    }),
    [overviewDays, rankingDays, rankingLimit, trendDays],
  )

  const { overviewQuery, trendQuery, topQueriesQuery, intentDistributionQuery } =
    useQAMetricsQueries(filters)

  const refreshAll = () => {
    void overviewQuery.refetch()
    void trendQuery.refetch()
    void topQueriesQuery.refetch()
    void intentDistributionQuery.refetch()
  }

  const trendPoints = trendQuery.data?.points ?? trendQuery.data?.trend30d ?? []
  const isFetching =
    overviewQuery.isFetching ||
    trendQuery.isFetching ||
    topQueriesQuery.isFetching ||
    intentDistributionQuery.isFetching

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h3 className="text-2xl font-semibold text-foreground">QA 统计</h3>
          <p className="mt-2 text-sm text-muted-foreground">即时指标</p>
        </div>
        <div className="flex flex-wrap items-end gap-3">
          <label className="w-28 space-y-1 text-sm">
            <span className="font-medium text-foreground">概览天数</span>
            <Input
              value={overviewDays}
              inputMode="numeric"
              onChange={(event) => setOverviewDays(event.target.value)}
            />
          </label>
          <Button
            type="button"
            onClick={refreshAll}
            disabled={isFetching}
            className="bg-primary text-primary-foreground transition-all duration-200 hover:bg-primary/90 hover:scale-[1.02] hover:shadow-lg active:scale-[0.98]"
          >
            刷新
          </Button>
        </div>
      </div>

      {/* Row 1: Instant stat cards */}
      <section>
        {overviewQuery.isError ? (
          <SectionState
            tone="error"
            message={`概览指标加载失败：${getErrorMessage(overviewQuery.error)}`}
          />
        ) : overviewQuery.isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="h-20 rounded-lg border border-border bg-card p-4">
                <SkeletonBlock className="mb-2 h-3 w-16" />
                <SkeletonBlock className="h-6 w-24" />
              </div>
            ))}
          </div>
        ) : overviewQuery.data ? (
          <StatCards overview={overviewQuery.data} />
        ) : (
          <SectionState tone="empty" message="暂无概览指标。" />
        )}
      </section>

      {/* Row 2: Volume rose chart (full width) */}
      <section className="rounded-lg border border-border bg-card p-5">
        <h4 className="mb-1 text-lg font-semibold text-foreground">容量指标</h4>
        <p className="mb-4 text-sm text-muted-foreground">总问答、问题、知识库、文档存量</p>
        {overviewQuery.isError ? (
          <SectionState
            tone="error"
            message={`容量指标加载失败：${getErrorMessage(overviewQuery.error)}`}
          />
        ) : overviewQuery.isLoading ? (
          <ChartSkeleton height={300} />
        ) : overviewQuery.data ? (
          <MetricsRoseChart overview={overviewQuery.data} />
        ) : (
          <SectionState tone="empty" message="暂无容量指标。" />
        )}
      </section>

      {/* Row 3: Trend pictorial bar + Intent gap pie */}
      <section className="grid gap-4 xl:grid-cols-2">
        <div className="space-y-4 rounded-lg border border-border bg-card p-5">
          <div className="flex flex-wrap items-end justify-between gap-3">
            <div>
              <h4 className="text-lg font-semibold text-foreground">问答趋势</h4>
              <p className="mt-1 text-sm text-muted-foreground">按日期展示问答数量</p>
            </div>
            <label className="w-28 space-y-1 text-sm">
              <span className="font-medium text-foreground">天数</span>
              <Input
                value={trendDays}
                inputMode="numeric"
                onChange={(event) => setTrendDays(event.target.value)}
              />
            </label>
          </div>
          {trendQuery.isError ? (
            <SectionState
              tone="error"
              message={`趋势加载失败：${getErrorMessage(trendQuery.error)}`}
            />
          ) : trendQuery.isLoading ? (
            <ChartSkeleton height={300} />
          ) : trendPoints.length === 0 ? (
            <SectionState tone="empty" message="当前窗口内暂无趋势数据。" />
          ) : (
            <TrendChart points={trendPoints} />
          )}
        </div>

        <div className="space-y-4 rounded-lg border border-border bg-card p-5">
          <div>
            <h4 className="text-lg font-semibold text-foreground">意图分布</h4>
            <p className="mt-1 text-sm text-muted-foreground">引导线标注占比</p>
          </div>
          {intentDistributionQuery.isError ? (
            <SectionState
              tone="error"
              message={`意图分布加载失败：${getErrorMessage(intentDistributionQuery.error)}`}
            />
          ) : intentDistributionQuery.isLoading ? (
            <ChartSkeleton height={320} />
          ) : (intentDistributionQuery.data ?? []).length === 0 ? (
            <SectionState tone="empty" message="当前窗口内暂无意图分布数据。" />
          ) : (
            <IntentGapPie items={intentDistributionQuery.data ?? []} />
          )}
        </div>
      </section>

      {/* Row 4: Top queries */}
      <section className="space-y-4 rounded-lg border border-border bg-card p-5">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h4 className="text-lg font-semibold text-foreground">热门问题</h4>
            <p className="mt-1 text-sm text-muted-foreground">按提问次数排序展示。</p>
          </div>
          <div className="flex gap-3">
            <label className="w-28 space-y-1 text-sm">
              <span className="font-medium text-foreground">天数</span>
              <Input
                value={rankingDays}
                inputMode="numeric"
                onChange={(event) => setRankingDays(event.target.value)}
              />
            </label>
            <label className="w-28 space-y-1 text-sm">
              <span className="font-medium text-foreground">条数</span>
              <Input
                value={rankingLimit}
                inputMode="numeric"
                onChange={(event) => setRankingLimit(event.target.value)}
              />
            </label>
          </div>
        </div>

        {topQueriesQuery.isError ? (
          <SectionState
            tone="error"
            message={`热门问题加载失败：${getErrorMessage(topQueriesQuery.error)}`}
          />
        ) : topQueriesQuery.isLoading ? (
          <ChartSkeleton height={240} />
        ) : (topQueriesQuery.data ?? []).length === 0 ? (
          <SectionState tone="empty" message="当前窗口内暂无热门问题。" />
        ) : (
          <TopQueriesTable queries={topQueriesQuery.data ?? []} />
        )}

        <div className="text-xs text-muted-foreground">
          当前参数：overview {formatNumber(filters.overviewDays)} 天，trend{' '}
          {formatNumber(filters.trendDays)} 天，ranking {formatNumber(filters.rankingDays)} 天。
        </div>
      </section>
    </div>
  )
}
