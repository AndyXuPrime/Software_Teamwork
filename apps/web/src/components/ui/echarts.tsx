'use client'

import { BarChart, LineChart, PictorialBarChart, PieChart } from 'echarts/charts'
import {
  DataZoomComponent,
  GridComponent,
  LegendComponent,
  TitleComponent,
  TooltipComponent,
} from 'echarts/components'
import * as echarts from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import * as React from 'react'

// Tree-shakeable: only register what we use
echarts.use([
  CanvasRenderer,
  LineChart,
  BarChart,
  PictorialBarChart,
  PieChart,
  GridComponent,
  TooltipComponent,
  LegendComponent,
  TitleComponent,
  DataZoomComponent,
])

type EChartsWrapperProps = {
  option: echarts.EChartsCoreOption
  className?: string
  style?: React.CSSProperties
  theme?: string | object
  onChartReady?: (instance: echarts.ECharts) => void
}

function EChartsWrapper({ option, className, style, theme, onChartReady }: EChartsWrapperProps) {
  const containerRef = React.useRef<HTMLDivElement | null>(null)
  const chartRef = React.useRef<echarts.ECharts | null>(null)

  React.useEffect(() => {
    if (!containerRef.current) return

    const instance = echarts.init(containerRef.current, theme)
    chartRef.current = instance
    instance.setOption(option, true)
    onChartReady?.(instance)

    const handleResize = () => instance.resize()
    window.addEventListener('resize', handleResize)

    return () => {
      window.removeEventListener('resize', handleResize)
      instance.dispose()
      chartRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  React.useEffect(() => {
    if (chartRef.current) {
      chartRef.current.setOption(option, true)
    }
  }, [option])

  return (
    <div
      ref={containerRef}
      className={className}
      style={{ width: '100%', minHeight: 300, ...style }}
    />
  )
}

export { EChartsWrapper }
export type { EChartsWrapperProps }
