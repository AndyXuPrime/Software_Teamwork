import { useRouter } from '@tanstack/react-router'
import { Loader2, LogIn, UserPlus } from 'lucide-react'
import { type FormEvent, useEffect, useMemo, useState } from 'react'
import { z } from 'zod'

import { ApiError } from '@/api/client'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useAuthStore } from '@/stores/auth-store'

const authSchema = z.object({
  username: z.string().trim().min(1, '请输入用户名'),
  password: z.string().min(1, '请输入密码'),
})

type AuthMode = 'login' | 'register'

function toErrorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return '认证失败，请稍后重试'
}

export function LoginPage() {
  const router = useRouter()
  const login = useAuthStore((state) => state.login)
  const register = useAuthStore((state) => state.register)
  const status = useAuthStore((state) => state.status)
  const storeError = useAuthStore((state) => state.error)
  const [mode, setMode] = useState<AuthMode>('login')
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [formError, setFormError] = useState<string | null>(null)

  const isSubmitting = status === 'restoring'
  const title = mode === 'login' ? '登录系统' : '创建账号'
  const submitLabel = mode === 'login' ? '登录' : '注册并登录'
  const error = formError ?? storeError

  const modeHint = useMemo(
    () =>
      mode === 'login'
        ? '使用 Gateway 会话接口创建当前登录会话。'
        : '注册接口会返回新用户和当前会话。',
    [mode],
  )

  useEffect(() => {
    if (status === 'authenticated') {
      void router.navigate({ to: '/' })
    }
  }, [router, status])

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setFormError(null)

    const parsed = authSchema.safeParse({ username, password })
    if (!parsed.success) {
      setFormError(parsed.error.issues[0]?.message ?? '表单校验失败')
      return
    }

    try {
      if (mode === 'login') {
        await login(parsed.data)
      } else {
        await register(parsed.data)
      }
      await router.navigate({ to: '/' })
    } catch (caught) {
      setFormError(toErrorMessage(caught))
    }
  }

  return (
    <main className="grid min-h-full place-items-center bg-background px-4 py-8 text-foreground">
      <section className="w-full max-w-sm rounded-lg border border-border bg-card p-6 shadow-sm">
        <div className="mb-6 space-y-1">
          <p className="text-xs font-medium text-muted-foreground">电力行业知识助手</p>
          <h1 className="text-xl font-semibold">{title}</h1>
          <p className="text-sm text-muted-foreground">{modeHint}</p>
        </div>

        <form className="space-y-4" onSubmit={handleSubmit}>
          <label className="block space-y-1.5 text-sm font-medium">
            <span>用户名</span>
            <Input
              autoComplete="username"
              disabled={isSubmitting}
              value={username}
              onChange={(event) => setUsername(event.target.value)}
            />
          </label>

          <label className="block space-y-1.5 text-sm font-medium">
            <span>密码</span>
            <Input
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              disabled={isSubmitting}
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </label>

          {error && (
            <div className="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              {error}
            </div>
          )}

          <Button className="w-full" disabled={isSubmitting} type="submit">
            {isSubmitting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : mode === 'login' ? (
              <LogIn className="size-4" />
            ) : (
              <UserPlus className="size-4" />
            )}
            {submitLabel}
          </Button>
        </form>

        <div className="mt-4 text-center text-sm text-muted-foreground">
          {mode === 'login' ? '还没有账号？' : '已有账号？'}
          <button
            className="ml-1 font-medium text-primary hover:underline"
            disabled={isSubmitting}
            type="button"
            onClick={() => {
              setFormError(null)
              setMode((current) => (current === 'login' ? 'register' : 'login'))
            }}
          >
            {mode === 'login' ? '创建账号' : '返回登录'}
          </button>
        </div>
      </section>
    </main>
  )
}
