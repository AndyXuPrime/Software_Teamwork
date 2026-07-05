import { Link, useRouter, useRouterState } from '@tanstack/react-router'
import {
  ChevronDown,
  ChevronRight,
  HelpCircle,
  Loader2,
  LogOut,
  RefreshCw,
  ShieldAlert,
  UserRound,
} from 'lucide-react'
import { type PropsWithChildren, type ReactNode, useEffect, useMemo, useRef, useState } from 'react'

import { apiClient } from '@/api/client'
import { AppVersionBadge } from '@/components/common/app-version-badge'
import { Button } from '@/components/ui/button'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import { adminShellAccess } from '@/lib/access'
import type { PermissionRequirement } from '@/lib/permissions'
import { canAccess } from '@/lib/permissions'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'
import { useChatStore } from '@/stores/chat-store'
import { usePageTransitionStore } from '@/stores/page-transition-store'

const pathLabels: Record<string, string> = {
  '/chat': '智能问答',
  '/qa': '智能问答',
  '/knowledge': '知识检索',
  '/reports': '报告生成',
  '/admin': '系统管理',
  '/profile': '个人资料',
  '/forbidden': '权限不足',
}

const navItems: Array<{
  label: string
  to: '/chat' | '/reports' | '/admin'
  requirement?: PermissionRequirement
}> = [
  { label: '问答', to: '/chat', requirement: { any: ['qa:use'] } },
  {
    label: '报告',
    to: '/reports',
    requirement: { any: ['report:read', 'report:write', 'reports:write'] },
  },
  {
    label: '管理',
    to: '/admin',
    requirement: adminShellAccess,
  },
]

const helpSections = [
  {
    title: '基础配置',
    steps: [
      '进入管理 / 模型管理，新增并启用用途为聊天的模型 Profile。',
      '进入管理 / QA / LLM 配置，选择聊天模型，测试连接后发布配置。',
      '进入管理 / 报告生成 / 文档模型配置，选择模型并发布文档生成配置。',
    ],
  },
  {
    title: '准备知识资料',
    steps: [
      '进入管理 / RAG 知识库 / 知识管理，新建知识库并选择文档类型和检索策略。',
      '进入文档管理上传文件，等待状态变为就绪后再用于检索、问答或报告引用。',
      '需要诊断检索效果时，先用知识检索或 QA 检索测试确认命中内容。',
    ],
  },
  {
    title: '开始问答',
    steps: [
      '进入问答页，新建或选择会话。',
      '按需选择知识库范围；不选择时使用当前 QA 配置或项目默认范围。',
      '可上传本次会话附件，输入问题后发送，等待流式回答完成。',
    ],
  },
  {
    title: '生成报告',
    steps: [
      '进入管理 / 报告生成 / 模板素材，上传 DOCX 模板，按需上传专业素材。',
      '进入报告页，填写报告名称、类型、模板、主题和生成要求。',
      '创建草稿并生成大纲，编辑后保存大纲，可选择知识库引用再生成正文。',
      '检查或修改章节正文，创建 DOCX 文件资源，就绪后下载文件。',
    ],
  },
] as const

function FullPageState({
  action,
  children,
  title,
}: PropsWithChildren<{ action?: ReactNode; title: string }>) {
  return (
    <div className="flex h-full items-center justify-center bg-background p-6 text-foreground">
      <section className="w-full max-w-md rounded-lg border border-border bg-card p-6 text-center shadow-sm">
        <h1 className="text-lg font-semibold">{title}</h1>
        <div className="mt-2 text-sm text-muted-foreground">{children}</div>
        {action && <div className="mt-5">{action}</div>}
      </section>
    </div>
  )
}

function HelpFlowSection({
  defaultOpen = false,
  steps,
  title,
}: {
  defaultOpen?: boolean
  steps: readonly string[]
  title: string
}) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <Collapsible
      className="overflow-hidden rounded-lg border border-border bg-card"
      open={open}
      onOpenChange={setOpen}
    >
      <CollapsibleTrigger className="flex min-h-11 w-full items-center justify-between gap-3 px-3 py-2.5 text-left transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/40">
        <span className="flex min-w-0 items-center gap-2">
          <span className="truncate font-medium text-foreground">{title}</span>
          <span className="shrink-0 rounded-md bg-muted px-1.5 py-0.5 text-xs text-muted-foreground">
            {steps.length} 步
          </span>
        </span>
        <ChevronDown
          aria-hidden="true"
          className={cn(
            'size-4 shrink-0 text-muted-foreground transition-transform duration-150',
            open && 'rotate-180',
          )}
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="px-3 pb-3">
        <ol className="list-decimal space-y-1.5 pl-5 text-sm leading-6 text-muted-foreground">
          {steps.map((step) => (
            <li key={step}>{step}</li>
          ))}
        </ol>
      </CollapsibleContent>
    </Collapsible>
  )
}

export function AppLayout({ children }: PropsWithChildren) {
  const router = useRouter()
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const isAdminRoute = pathname.startsWith('/admin')
  const pageTransitionKey = isAdminRoute ? '/admin' : pathname
  const [helpOpen, setHelpOpen] = useState(false)
  const currentLabel =
    Object.entries(pathLabels).find(([key]) => pathname.startsWith(key))?.[1] ?? '首页'
  const user = useAuthStore((state) => state.user)
  const status = useAuthStore((state) => state.status)
  const error = useAuthStore((state) => state.error)
  const restoreSession = useAuthStore((state) => state.restoreSession)
  const qaUnreadCompletion = useChatStore((state) => state.qaUnreadCompletion)
  const setQaChatVisible = useChatStore((state) => state.setQaChatVisible)

  // Page transition entrance
  useEffect(() => {
    usePageTransitionStore.getState().reveal()
  }, [])

  useEffect(() => {
    setQaChatVisible(pathname.startsWith('/chat'))
  }, [pathname, setQaChatVisible])

  // ── Nav slider position ──
  const visibleNavItems = useMemo(
    () => navItems.filter((item) => canAccess(user, item.requirement)),
    [user],
  )
  const navRefs = useRef<Record<string, HTMLAnchorElement | null>>({})
  const [sliderStyle, setSliderStyle] = useState<{ left: number; width: number }>({
    left: 0,
    width: 0,
  })

  useEffect(() => {
    const id = pathname.startsWith('/chat')
      ? '/chat'
      : pathname.startsWith('/reports')
        ? '/reports'
        : '/admin'
    const raf = requestAnimationFrame(() => {
      const el = navRefs.current[id]
      if (el) {
        const parentRect = el.parentElement!.getBoundingClientRect()
        const elRect = el.getBoundingClientRect()
        const newLeft = elRect.left - parentRect.left
        const newWidth = elRect.width
        setSliderStyle((prev) =>
          prev.left === newLeft && prev.width === newWidth
            ? prev
            : { left: newLeft, width: newWidth },
        )
      }
    })
    return () => cancelAnimationFrame(raf)
  }, [pathname, visibleNavItems])

  if (status === 'restoring' || status === 'idle') {
    return (
      <FullPageState title="正在恢复会话">
        <span className="inline-flex items-center gap-2">
          <Loader2 className="size-4 animate-spin" />
          正在读取当前用户信息
        </span>
      </FullPageState>
    )
  }

  if (status === 'error') {
    return (
      <FullPageState
        action={
          <Button variant="outline" onClick={() => void restoreSession()}>
            <RefreshCw className="size-4" />
            重试
          </Button>
        }
        title="会话恢复失败"
      >
        {error ?? '无法读取当前用户，请稍后重试。'}
      </FullPageState>
    )
  }

  const handleLogout = () => {
    const token = apiClient.getToken()
    useAuthStore.getState().clearSession()
    void router.navigate({ to: '/login' })
    // Fire-and-forget: end the server session with captured token
    if (token && token !== 'dev-token-bypass') {
      void fetch(`${apiClient.baseUrl}/sessions/current`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      }).catch(() => {
        /* best-effort */
      })
    }
  }

  return (
    <div className="flex h-full flex-col bg-background text-foreground">
      <header className="flex h-14 items-center justify-between border-b border-primary/30 bg-primary/5 px-6">
        <div className="flex min-w-0 items-center gap-3">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-primary text-sm font-semibold text-primary-foreground">
            电
          </div>
          <span className="truncate text-sm text-primary">电力行业知识助手</span>
          <span className="truncate text-sm font-semibold">{currentLabel}</span>
          <AppVersionBadge className="hidden shrink-0 lg:inline-flex" />
        </div>

        <nav
          aria-label="主导航"
          className="relative flex items-center gap-1 rounded-lg border border-border/80 bg-muted/60 p-1 text-sm shadow-inner"
        >
          {/* Sliding pill — only visible when on a main nav item, not /profile */}
          {visibleNavItems.some((item) => pathname.startsWith(item.to)) && (
            <div
              aria-hidden
              className="absolute top-1 h-[calc(100%-8px)] rounded-md bg-background shadow-sm transition-all duration-300 ease-out"
              style={{ left: sliderStyle.left, width: sliderStyle.width }}
            />
          )}
          {visibleNavItems.map((item) => (
            <Link
              key={item.to}
              to={item.to}
              ref={(el) => {
                navRefs.current[item.to] = el as HTMLAnchorElement | null
              }}
              className="relative z-10 rounded-md px-3 py-1.5 transition-colors hover:text-foreground"
              activeProps={{ className: 'text-foreground font-medium' }}
              inactiveProps={{ className: 'text-muted-foreground' }}
            >
              {item.label}
              {item.to === '/chat' && qaUnreadCompletion && (
                <span
                  aria-hidden
                  className="absolute right-1 top-1 h-2 w-2 rounded-full bg-destructive ring-2 ring-background"
                  data-testid="qa-unread-dot"
                />
              )}
            </Link>
          ))}
        </nav>

        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Button
            aria-label="打开帮助"
            size="icon-sm"
            title="帮助"
            type="button"
            variant="ghost"
            onClick={() => setHelpOpen(true)}
          >
            <HelpCircle />
          </Button>
          <Link
            aria-label="打开个人资料"
            className={cn(
              'hidden max-w-40 items-center gap-1 rounded-md border border-border bg-background px-2 py-1 shadow-sm transition-all hover:border-ring/30 hover:shadow hover:text-foreground sm:inline-flex',
              pathname === '/profile' &&
                'border-primary/40 bg-primary/10 text-primary shadow-none font-medium',
            )}
            to="/profile"
          >
            <UserRound className="size-3.5" />
            <span className="truncate">{user?.displayName || user?.username || '未登录'}</span>
            <ChevronRight className="size-3 shrink-0 opacity-50" />
          </Link>
          {user && user.roles.length > 0 && (
            <span className="hidden rounded-md bg-muted px-2 py-1 sm:inline">
              {user.roles.join(', ')}
            </span>
          )}
          {!visibleNavItems.length && (
            <span className="inline-flex items-center gap-1 text-destructive">
              <ShieldAlert className="size-3.5" />
              无可用菜单
            </span>
          )}
          <Button
            aria-label="退出登录"
            size="icon-sm"
            type="button"
            variant="ghost"
            onClick={handleLogout}
          >
            <LogOut />
          </Button>
        </div>
      </header>

      <main
        key={pageTransitionKey}
        className={cn(
          'flex-1 overflow-y-auto overflow-x-hidden',
          !isAdminRoute && 'page-enter-right',
        )}
        style={{ scrollbarGutter: 'stable' }}
      >
        {children}
      </main>

      <Dialog open={helpOpen} onOpenChange={setHelpOpen}>
        <DialogContent className="max-h-[calc(100vh-2rem)] sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>Helpme</DialogTitle>
            <DialogDescription>按当前页面入口完成配置和使用。</DialogDescription>
          </DialogHeader>
          <ScrollArea
            className="h-[34rem] max-h-[68vh]"
            viewportClassName="pr-3"
            aria-label="Helpme 引导流程"
          >
            <div className="grid gap-2 pb-1">
              {helpSections.map((section, index) => (
                <HelpFlowSection
                  key={section.title}
                  defaultOpen={index === 0}
                  steps={section.steps}
                  title={section.title}
                />
              ))}
            </div>
          </ScrollArea>
        </DialogContent>
      </Dialog>
    </div>
  )
}
