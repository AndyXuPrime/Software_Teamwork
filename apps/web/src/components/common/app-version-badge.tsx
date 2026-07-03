import { AlertTriangle, CheckCircle2, Loader2, RefreshCw, WifiOff } from 'lucide-react'
import { useCallback, useState } from 'react'

import { badgeVariants } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Popover,
  PopoverContent,
  PopoverDescription,
  PopoverHeader,
  PopoverTitle,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  APP_UPDATE_COMMAND,
  appCommitShortSha,
  type AppFreshnessResult,
  appVersionLabel,
  checkUpstreamDevelopFreshness,
  formatAppVersion,
  formatCommitLabel,
} from '@/lib/app-version'
import { cn } from '@/lib/utils'

type AppVersionBadgeProps = {
  className?: string
  checkLatest?: typeof checkUpstreamDevelopFreshness
  version?: string | null
}

type FreshnessState =
  | { status: 'idle' }
  | { status: 'checking' }
  | { result: AppFreshnessResult; status: 'result' }
  | { message: string; status: 'error' }

function freshnessTitle(state: FreshnessState) {
  if (state.status === 'checking') return '正在检查 upstream/develop'
  if (state.status === 'error') return '检查失败'
  if (state.status === 'result') {
    if (state.result.status === 'current') return '已包含最新 develop'
    if (state.result.status === 'different') {
      return `当前构建落后 develop ${state.result.commitsBehind} 个提交`
    }
    return '无法判断当前构建状态'
  }

  return '点击检查 upstream/develop'
}

function freshnessDescription(state: FreshnessState) {
  if (state.status === 'checking') return '正在联网读取 GitHub 最新提交。'
  if (state.status === 'error') return state.message
  if (state.status === 'result') {
    if (state.result.status === 'current') {
      if (state.result.commitsAhead > 0) {
        return `当前构建不落后于 upstream/develop，并包含 ${state.result.commitsAhead} 个本地或功能分支提交。`
      }

      return '当前构建提交和 upstream/develop 最新提交一致。'
    }
    if (state.result.status === 'different') {
      return '当前构建提交和 upstream/develop 最新提交不同，建议在终端同步最新 develop 后重新启动前端。'
    }

    return '当前构建没有可用提交号，无法和 upstream/develop 精确比对。'
  }

  return '只检查远端状态，不会自动更新本地代码。'
}

function FreshnessIcon({ state }: { state: FreshnessState }) {
  if (state.status === 'checking') return <Loader2 className="size-3.5 animate-spin" />
  if (state.status === 'error') return <WifiOff className="size-3.5 text-destructive" />
  if (state.status === 'result' && state.result.status === 'current') {
    return <CheckCircle2 className="size-3.5 text-primary" />
  }
  if (state.status === 'result') return <AlertTriangle className="size-3.5 text-destructive" />

  return <RefreshCw className="size-3.5" />
}

export function AppVersionBadge({
  checkLatest = checkUpstreamDevelopFreshness,
  className,
  version = appVersionLabel,
}: AppVersionBadgeProps) {
  const label = formatAppVersion(version)
  const [state, setState] = useState<FreshnessState>({ status: 'idle' })

  const handleCheck = useCallback(async () => {
    setState({ status: 'checking' })
    try {
      const result = await checkLatest()
      setState({ result, status: 'result' })
    } catch (error) {
      setState({
        message: error instanceof Error ? error.message : '无法连接 GitHub',
        status: 'error',
      })
    }
  }, [checkLatest])

  const isDifferent = state.status === 'result' && state.result.status === 'different'

  return (
    <Popover>
      <PopoverTrigger
        aria-label={`前端版本 ${label}，点击检查 upstream/develop 更新状态`}
        className={cn(
          badgeVariants({ variant: 'outline' }),
          'cursor-pointer border-primary/25 bg-background/70 font-mono text-[0.68rem] text-muted-foreground hover:bg-muted hover:text-foreground',
          isDifferent && 'border-destructive/40 text-destructive hover:text-destructive',
          className,
        )}
        title={`前端版本 ${label}，点击检查 upstream/develop 更新状态`}
        onClick={() => void handleCheck()}
      >
        {label}
      </PopoverTrigger>
      <PopoverContent align="end" className="w-80 gap-3 text-xs">
        <PopoverHeader>
          <PopoverTitle className="flex items-center gap-2 text-sm">
            <FreshnessIcon state={state} />
            {freshnessTitle(state)}
          </PopoverTitle>
          <PopoverDescription>{freshnessDescription(state)}</PopoverDescription>
        </PopoverHeader>

        <dl className="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-muted-foreground">
          <dt>当前版本</dt>
          <dd className="font-mono text-foreground">{label}</dd>
          <dt>当前提交</dt>
          <dd className="font-mono text-foreground">{appCommitShortSha}</dd>
          {state.status === 'result' && (
            <>
              <dt>develop</dt>
              <dd className="font-mono text-foreground">
                {state.result.latestUrl ? (
                  <a
                    className="underline-offset-4 hover:underline"
                    href={state.result.latestUrl}
                    rel="noreferrer"
                    target="_blank"
                  >
                    {formatCommitLabel(state.result.latestSha)}
                  </a>
                ) : (
                  formatCommitLabel(state.result.latestSha)
                )}
              </dd>
              <dt>落后提交</dt>
              <dd className="font-mono text-foreground">{state.result.commitsBehind} 个</dd>
            </>
          )}
        </dl>

        {isDifferent && (
          <div className="rounded-md border border-destructive/20 bg-destructive/5 p-2">
            <div className="font-medium text-destructive">建议更新</div>
            <code className="mt-1 block break-all font-mono text-[0.68rem] text-foreground">
              {APP_UPDATE_COMMAND}
            </code>
          </div>
        )}

        <Button
          className="w-full"
          disabled={state.status === 'checking'}
          size="sm"
          type="button"
          variant="outline"
          onClick={() => void handleCheck()}
        >
          <RefreshCw className={cn('size-3.5', state.status === 'checking' && 'animate-spin')} />
          重新检查
        </Button>
      </PopoverContent>
    </Popover>
  )
}
