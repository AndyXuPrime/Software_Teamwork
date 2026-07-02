import { Check, Moon, PaintBucket, Radius, Sun, Type } from 'lucide-react'

import type { ColorKey, ColorPreset } from '@/hooks'
import { FONT_SIZE_SCALES, PRIMARY_COLORS } from '@/hooks'
import { cn } from '@/lib/utils'
import { useThemeStore } from '@/stores/theme-store'

const RADIUS_OPTIONS = [
  { value: 0, desc: '直角' },
  { value: 0.375, desc: '小' },
  { value: 0.625, desc: '默认' },
  { value: 1, desc: '大' },
] as const

const FONT_SIZE_OPTIONS = [
  { value: 'small', label: '小' },
  { value: 'medium', label: '中' },
  { value: 'large', label: '大' },
] as const

export function AppearanceSettings() {
  const mode = useThemeStore((state) => state.mode)
  const setMode = useThemeStore((state) => state.setMode)
  const primaryColor = useThemeStore((state) => state.primaryColor)
  const setPrimaryColor = useThemeStore((state) => state.setPrimaryColor)
  const radius = useThemeStore((state) => state.radius)
  const setRadius = useThemeStore((state) => state.setRadius)
  const fontSize = useThemeStore((state) => state.fontSize)
  const setFontSize = useThemeStore((state) => state.setFontSize)

  return (
    <section
      aria-labelledby="appearance-settings-title"
      className="rounded-lg border border-border bg-card"
    >
      <div className="border-b border-border px-4 py-3">
        <h2 id="appearance-settings-title" className="text-sm font-semibold text-foreground">
          界面外观
        </h2>
        <p className="mt-1 text-xs text-muted-foreground">
          调整当前浏览器内的主题、主题色、圆角和字体大小。
        </p>
      </div>

      <div className="space-y-8 p-4">
        <section aria-labelledby="appearance-mode-title">
          <div className="mb-3 flex items-center gap-2">
            <Sun aria-hidden="true" className="size-4 text-muted-foreground" />
            <h3 id="appearance-mode-title" className="text-base font-semibold text-foreground">
              外观模式
            </h3>
          </div>

          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <button
              type="button"
              aria-pressed={mode === 'light'}
              onClick={() => setMode('light')}
              className={cn(
                'cursor-pointer rounded-lg border-2 p-4 text-left transition-all duration-200 hover:scale-[1.01]',
                mode === 'light'
                  ? 'border-primary ring-2 ring-ring/30'
                  : 'border-border hover:border-muted-foreground/30',
              )}
            >
              <div className="mb-3 overflow-hidden rounded-lg border border-border bg-white">
                <div className="h-2 bg-muted" />
                <div className="space-y-1.5 px-2 py-2">
                  <div className="h-1.5 w-3/4 rounded bg-muted-foreground/20" />
                  <div className="h-1.5 w-1/2 rounded bg-muted-foreground/15" />
                  <div className="h-1.5 w-2/3 rounded bg-muted-foreground/15" />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Sun aria-hidden="true" className="size-4 text-foreground" />
                <span className="text-sm font-medium text-foreground">浅色模式</span>
                {mode === 'light' && (
                  <Check aria-hidden="true" className="ml-auto size-4 text-primary" />
                )}
              </div>
            </button>

            <button
              type="button"
              aria-pressed={mode === 'dark'}
              onClick={() => setMode('dark')}
              className={cn(
                'cursor-pointer rounded-lg border-2 p-4 text-left transition-all duration-200 hover:scale-[1.01]',
                mode === 'dark'
                  ? 'border-primary ring-2 ring-ring/30'
                  : 'border-border hover:border-muted-foreground/30',
              )}
            >
              <div className="mb-3 overflow-hidden rounded-lg border border-border bg-[oklch(0.205_0_0)]">
                <div className="h-2 bg-[oklch(0.269_0_0)]" />
                <div className="space-y-1.5 px-2 py-2">
                  <div className="h-1.5 w-3/4 rounded bg-white/20" />
                  <div className="h-1.5 w-1/2 rounded bg-white/10" />
                  <div className="h-1.5 w-2/3 rounded bg-white/10" />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Moon aria-hidden="true" className="size-4 text-foreground" />
                <span className="text-sm font-medium text-foreground">深色模式</span>
                {mode === 'dark' && (
                  <Check aria-hidden="true" className="ml-auto size-4 text-primary" />
                )}
              </div>
            </button>
          </div>
        </section>

        <section aria-labelledby="appearance-primary-title">
          <div className="mb-3 flex items-center gap-2">
            <PaintBucket aria-hidden="true" className="size-4 text-muted-foreground" />
            <h3 id="appearance-primary-title" className="text-base font-semibold text-foreground">
              主题色
            </h3>
          </div>

          <div className="grid grid-cols-2 gap-2 xs:grid-cols-3 sm:grid-cols-5">
            {(Object.entries(PRIMARY_COLORS) as [ColorKey, ColorPreset][]).map(([key, preset]) => {
              const isSelected = primaryColor === key
              return (
                <button
                  key={key}
                  type="button"
                  aria-pressed={isSelected}
                  onClick={() => setPrimaryColor(key)}
                  className="flex cursor-pointer flex-col items-center gap-1.5 rounded-lg py-2 transition-all duration-200 hover:scale-[1.01] hover:bg-muted/50"
                >
                  <span
                    className={cn(
                      'relative flex size-9 items-center justify-center rounded-full border-2 transition-all',
                      isSelected ? 'border-primary ring-2 ring-ring/20' : 'border-transparent',
                    )}
                    style={{ backgroundColor: preset.light }}
                  >
                    {isSelected && (
                      <Check
                        aria-hidden="true"
                        className="size-4"
                        style={{
                          color: key === 'yellow' ? 'oklch(0.205 0 0)' : 'oklch(0.985 0 0)',
                        }}
                      />
                    )}
                  </span>
                  <span
                    className={cn(
                      'text-xs transition-colors',
                      isSelected ? 'font-medium text-foreground' : 'text-muted-foreground',
                    )}
                  >
                    {preset.label}
                  </span>
                </button>
              )
            })}
          </div>
        </section>

        <section aria-labelledby="appearance-radius-title">
          <div className="mb-3 flex items-center gap-2">
            <Radius aria-hidden="true" className="size-4 text-muted-foreground" />
            <h3 id="appearance-radius-title" className="text-base font-semibold text-foreground">
              圆角
            </h3>
          </div>

          <div className="flex flex-wrap items-start gap-4">
            <div className="flex rounded-lg border border-border bg-muted/50 p-0.5">
              {RADIUS_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={radius === option.value}
                  onClick={() => setRadius(option.value)}
                  className={cn(
                    'cursor-pointer rounded-md px-3 py-1.5 text-sm font-medium transition-all',
                    radius === option.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {option.desc}
                </button>
              ))}
            </div>

            <div className="flex min-w-0 flex-1 items-center gap-3">
              <div
                className="flex h-16 w-28 items-center justify-center border-2 border-dashed border-border bg-muted/30 text-xs text-muted-foreground transition-all"
                style={{ borderRadius: `${radius}rem` }}
              >
                预览
              </div>
              <span className="text-xs text-muted-foreground tabular-nums">{radius}rem</span>
            </div>
          </div>
        </section>

        <section aria-labelledby="appearance-font-size-title">
          <div className="mb-3 flex items-center gap-2">
            <Type aria-hidden="true" className="size-4 text-muted-foreground" />
            <h3 id="appearance-font-size-title" className="text-base font-semibold text-foreground">
              字体大小
            </h3>
          </div>

          <div className="flex flex-wrap items-start gap-4">
            <div className="flex rounded-lg border border-border bg-muted/50 p-0.5">
              {FONT_SIZE_OPTIONS.map((option) => (
                <button
                  key={option.value}
                  type="button"
                  aria-pressed={fontSize === option.value}
                  onClick={() => setFontSize(option.value)}
                  className={cn(
                    'cursor-pointer rounded-md px-4 py-1.5 text-sm font-medium transition-all',
                    fontSize === option.value
                      ? 'bg-background text-foreground shadow-sm'
                      : 'text-muted-foreground hover:text-foreground',
                  )}
                >
                  {option.label}
                </button>
              ))}
            </div>

            <div className="flex min-w-0 flex-1 items-center rounded-lg border border-border bg-muted/30 px-4 py-3">
              <p
                className="leading-relaxed text-foreground transition-all"
                style={{ fontSize: `calc(1rem * ${FONT_SIZE_SCALES[fontSize]})` }}
              >
                预览文字 Preview
              </p>
            </div>
          </div>
        </section>
      </div>
    </section>
  )
}
