import { useRouter } from '@tanstack/react-router'
import { KeyRound, Loader2, LogOut } from 'lucide-react'
import { type FormEvent, useEffect, useState } from 'react'

import { changeCurrentUserPassword } from '@/api/auth'
import { InlineNotice } from '@/components/common'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  formatUserManagementError,
  type PasswordChangeForm,
  validatePasswordChangeForm,
} from '@/features/auth'
import { useAuthStore } from '@/stores/auth-store'
import { usePageTransitionStore } from '@/stores/page-transition-store'

const EMPTY_FORM: PasswordChangeForm = {
  currentPassword: '',
  newPassword: '',
  newPasswordConfirmation: '',
}

export function PasswordChangeRequiredPage() {
  const router = useRouter()
  const user = useAuthStore((state) => state.user)
  const logout = useAuthStore((state) => state.logout)
  const [form, setForm] = useState<PasswordChangeForm>(EMPTY_FORM)
  const [error, setError] = useState<string | null>(null)
  const [isSubmitting, setSubmitting] = useState(false)
  const [isLoggingOut, setLoggingOut] = useState(false)

  useEffect(() => {
    usePageTransitionStore.getState().reveal()
  }, [])

  const updateField = <K extends keyof PasswordChangeForm>(
    field: K,
    value: PasswordChangeForm[K],
  ) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setError(null)
    const validation = validatePasswordChangeForm(form)
    if (!validation.valid || !validation.request) {
      setError(validation.message ?? '表单校验失败')
      return
    }

    setSubmitting(true)
    try {
      const profile = await changeCurrentUserPassword(validation.request)
      useAuthStore.setState({
        error: null,
        status: 'authenticated',
        user: profile,
        userName: profile.username,
      })
      await router.navigate({ to: '/' })
    } catch (caught) {
      setError(formatUserManagementError(caught, '修改密码失败'))
    } finally {
      setSubmitting(false)
    }
  }

  const handleLogout = async () => {
    setError(null)
    setLoggingOut(true)
    try {
      await logout()
    } catch (caught) {
      useAuthStore.getState().clearSession()
      console.warn('failed to revoke current session during forced password change logout', caught)
    } finally {
      setLoggingOut(false)
    }
    await router.navigate({ to: '/login' })
  }

  const isBusy = isSubmitting || isLoggingOut

  return (
    <main className="flex min-h-screen items-center justify-center bg-background p-4 text-foreground">
      <section className="w-full max-w-md rounded-lg border border-border bg-card p-6 shadow-sm">
        <div className="mb-5 flex items-center gap-3">
          <div className="flex size-10 items-center justify-center rounded-md bg-primary/10 text-primary">
            <KeyRound className="size-5" />
          </div>
          <div>
            <h1 className="text-lg font-semibold">需要修改临时密码</h1>
            <p className="mt-1 text-sm text-muted-foreground">
              {user?.username ? `${user.username}，` : ''}请先完成密码修改再进入系统。
            </p>
          </div>
        </div>

        {error && (
          <InlineNotice className="mb-4" variant="error">
            {error}
          </InlineNotice>
        )}

        <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
          <label className="block space-y-1.5 text-sm">
            <span className="text-muted-foreground">当前临时密码</span>
            <Input
              autoComplete="current-password"
              type="password"
              value={form.currentPassword}
              disabled={isBusy}
              onChange={(event) => updateField('currentPassword', event.target.value)}
            />
          </label>
          <label className="block space-y-1.5 text-sm">
            <span className="text-muted-foreground">新密码</span>
            <Input
              autoComplete="new-password"
              type="password"
              value={form.newPassword}
              disabled={isBusy}
              onChange={(event) => updateField('newPassword', event.target.value)}
            />
          </label>
          <label className="block space-y-1.5 text-sm">
            <span className="text-muted-foreground">确认新密码</span>
            <Input
              autoComplete="new-password"
              type="password"
              value={form.newPasswordConfirmation}
              disabled={isBusy}
              onChange={(event) => updateField('newPasswordConfirmation', event.target.value)}
            />
          </label>

          <Button className="w-full" disabled={isBusy} type="submit">
            {isSubmitting ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <KeyRound className="size-4" />
            )}
            修改密码
          </Button>
          <Button
            className="w-full"
            disabled={isBusy}
            type="button"
            variant="outline"
            onClick={() => void handleLogout()}
          >
            {isLoggingOut ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <LogOut className="size-4" />
            )}
            退出登录
          </Button>
        </form>
      </section>
    </main>
  )
}
