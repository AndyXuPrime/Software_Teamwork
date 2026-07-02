import { Loader2, RefreshCw, Save } from 'lucide-react'
import { useEffect, useState } from 'react'

import { InlineNotice, StateBlock } from '@/components/common'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  formatUserManagementError,
  type ProfileForm,
  roleLabel,
  statusLabel,
  useCurrentUserProfile,
  useUpdateCurrentUserProfile,
  validateProfileForm,
} from '@/features/auth'
import { AppearanceSettings } from '@/features/theme'
import { useAuthStore } from '@/stores/auth-store'

const EMPTY_FORM: ProfileForm = {
  displayName: '',
  email: '',
  phone: '',
}

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function profileToForm(profile: {
  displayName?: string
  email?: string | null
  phone?: string | null
}): ProfileForm {
  return {
    displayName: profile.displayName ?? '',
    email: profile.email ?? '',
    phone: profile.phone ?? '',
  }
}

export function ProfilePage() {
  const profileQuery = useCurrentUserProfile()
  const updateMutation = useUpdateCurrentUserProfile()
  const [form, setForm] = useState<ProfileForm>(EMPTY_FORM)
  const [notice, setNotice] = useState<{ text: string; type: 'error' | 'success' } | null>(null)

  useEffect(() => {
    if (profileQuery.data) {
      setForm(profileToForm(profileQuery.data))
    }
  }, [profileQuery.data])

  const updateField = <K extends keyof ProfileForm>(field: K, value: ProfileForm[K]) => {
    setForm((prev) => ({ ...prev, [field]: value }))
  }

  const handleSubmit = () => {
    setNotice(null)
    const validation = validateProfileForm(form)
    if (!validation.valid || !validation.request) {
      setNotice({ type: 'error', text: validation.message ?? '表单校验失败' })
      return
    }

    updateMutation.mutate(validation.request, {
      onSuccess: (profile) => {
        useAuthStore.setState({
          user: profile,
          userName: profile.username,
        })
        setForm(profileToForm(profile))
        setNotice({ type: 'success', text: '个人资料已更新' })
      },
      onError: (error) => {
        setNotice({
          type: 'error',
          text: formatUserManagementError(error, '更新个人资料失败'),
        })
      },
    })
  }

  if (profileQuery.isLoading) {
    return (
      <div className="p-6">
        <StateBlock
          title="正在加载个人资料"
          description="正在从 Gateway 读取当前用户资料。"
          variant="loading"
        />
      </div>
    )
  }

  if (profileQuery.isError) {
    return (
      <div className="p-6">
        <StateBlock
          action={
            <Button variant="outline" onClick={() => void profileQuery.refetch()}>
              <RefreshCw className="size-4" />
              重试
            </Button>
          }
          title="个人资料加载失败"
          description={formatUserManagementError(profileQuery.error, '个人资料加载失败')}
          variant="error"
        />
      </div>
    )
  }

  const profile = profileQuery.data
  if (!profile) {
    return (
      <div className="p-6">
        <StateBlock title="个人资料为空" description="当前响应没有包含用户资料。" variant="empty" />
      </div>
    )
  }

  const isSaving = updateMutation.isPending

  return (
    <div className="mx-auto flex w-full max-w-5xl flex-col gap-5 p-6">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">个人资料</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            查看账号身份、角色权限，并维护可选联系资料。
          </p>
        </div>
        <Button disabled={isSaving} onClick={handleSubmit}>
          {isSaving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
          保存资料
        </Button>
      </div>

      {notice && (
        <InlineNotice variant={notice.type === 'success' ? 'success' : 'error'}>
          {notice.text}
        </InlineNotice>
      )}

      <section className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
        <div className="rounded-lg border border-border bg-card">
          <div className="border-b border-border px-4 py-3">
            <h2 className="text-sm font-semibold text-foreground">账号身份</h2>
          </div>
          <dl className="grid gap-0 text-sm sm:grid-cols-2">
            {[
              ['用户 ID', profile.id],
              ['用户名', profile.username],
              ['状态', statusLabel(profile.status)],
              ['首次改密', profile.mustChangePassword ? '需要' : '不需要'],
              ['创建时间', formatDate(profile.createdAt)],
              ['更新时间', formatDate(profile.updatedAt)],
            ].map(([label, value]) => (
              <div key={label} className="border-b border-border px-4 py-3 last:border-b-0">
                <dt className="text-xs text-muted-foreground">{label}</dt>
                <dd className="mt-1 break-words font-medium text-foreground">{value}</dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="rounded-lg border border-border bg-card">
          <div className="border-b border-border px-4 py-3">
            <h2 className="text-sm font-semibold text-foreground">角色与权限</h2>
          </div>
          <div className="space-y-4 p-4">
            <div>
              <div className="mb-2 text-xs text-muted-foreground">角色</div>
              <div className="flex flex-wrap gap-1.5">
                {profile.roles.length ? (
                  profile.roles.map((role) => (
                    <Badge key={role} variant="secondary">
                      {roleLabel(role)}
                    </Badge>
                  ))
                ) : (
                  <span className="text-sm text-muted-foreground">无角色</span>
                )}
              </div>
            </div>
            <div>
              <div className="mb-2 text-xs text-muted-foreground">权限</div>
              <div className="flex max-h-44 flex-wrap gap-1.5 overflow-auto">
                {profile.permissions.length ? (
                  profile.permissions.map((permission) => (
                    <Badge key={permission} variant="outline">
                      {permission}
                    </Badge>
                  ))
                ) : (
                  <span className="text-sm text-muted-foreground">无权限</span>
                )}
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="rounded-lg border border-border bg-card">
        <div className="border-b border-border px-4 py-3">
          <h2 className="text-sm font-semibold text-foreground">可编辑资料</h2>
        </div>
        <div className="grid gap-4 p-4 sm:grid-cols-3">
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">显示名</span>
            <Input
              value={form.displayName}
              disabled={isSaving}
              onChange={(event) => updateField('displayName', event.target.value)}
            />
          </label>
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">邮箱</span>
            <Input
              type="email"
              value={form.email}
              disabled={isSaving}
              placeholder="可不填写"
              onChange={(event) => updateField('email', event.target.value)}
            />
          </label>
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">电话</span>
            <Input
              value={form.phone}
              disabled={isSaving}
              placeholder="可不填写"
              onChange={(event) => updateField('phone', event.target.value)}
            />
          </label>
        </div>
      </section>

      <AppearanceSettings />
    </div>
  )
}
