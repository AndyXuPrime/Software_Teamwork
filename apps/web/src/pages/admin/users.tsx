import { useNavigate, useSearch } from '@tanstack/react-router'
import {
  Edit3,
  Loader2,
  Plus,
  RefreshCw,
  RotateCcwKey,
  Search,
  ShieldAlert,
  UserCheck,
  UserX,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'

import { ConfirmDialog, InlineNotice, StateBlock, TableSkeleton } from '@/components/common'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  type AdminPasswordResetForm,
  type AdminRoleChangeForm,
  type AdminUserCreateForm,
  type AdminUsersSearch,
  canManageAdminUsers,
  DEFAULT_ADMIN_USERS_SEARCH,
  formatUserManagementError,
  getAssignableRoles,
  primaryManagedRole,
  roleLabel,
  statusLabel,
  useAdminUsers,
  useCreateAdminUser,
  USER_STATUS_OPTIONS,
  useResetAdminUserPassword,
  useUpdateAdminUser,
  validateCreateAdminUserForm,
  validatePasswordResetForm,
  validateRoleChangeForm,
} from '@/features/auth'
import type { AdminUser, ManagedUserRole, UserStatus } from '@/lib/types'
import { useAuthStore } from '@/stores/auth-store'

const EMPTY_CREATE_FORM: AdminUserCreateForm = {
  displayName: '',
  email: '',
  phone: '',
  role: 'standard',
  temporaryPassword: '',
  username: '',
}

const EMPTY_RESET_FORM: AdminPasswordResetForm = {
  temporaryPassword: '',
}

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function sanitizeSearch(search: AdminUsersSearch): AdminUsersSearch {
  return {
    page: search.page || DEFAULT_ADMIN_USERS_SEARCH.page,
    pageSize: search.pageSize || DEFAULT_ADMIN_USERS_SEARCH.pageSize,
    ...(search.username?.trim() ? { username: search.username.trim() } : {}),
    ...(search.role ? { role: search.role } : {}),
    ...(search.status ? { status: search.status } : {}),
  }
}

function isSelf(row: AdminUser, currentUserId?: string): boolean {
  return Boolean(currentUserId && row.id === currentUserId)
}

function canRowDisable(row: AdminUser, currentUserId?: string): boolean {
  return !isSelf(row, currentUserId) && row.status === 'active' && row.actions?.canDisable !== false
}

function canRowEnable(row: AdminUser, currentUserId?: string): boolean {
  return (
    !isSelf(row, currentUserId) && row.status === 'disabled' && row.actions?.canEnable !== false
  )
}

function canRowReset(row: AdminUser, currentUserId?: string): boolean {
  return !isSelf(row, currentUserId) && row.actions?.canResetPassword !== false
}

function canRowChangeRole(
  row: AdminUser,
  allowedRoles: readonly ManagedUserRole[],
  currentUserId?: string,
): boolean {
  if (isSelf(row, currentUserId) || row.actions?.canChangeRole === false) return false
  const rowRoles = row.manageableRoles?.length ? row.manageableRoles : allowedRoles
  return rowRoles.some((role) => allowedRoles.includes(role))
}

function statusBadgeVariant(
  status?: UserStatus,
): 'default' | 'destructive' | 'outline' | 'secondary' {
  if (status === 'active') return 'secondary'
  if (status === 'disabled') return 'destructive'
  return 'outline'
}

export function AdminUsersPage() {
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)
  const search = sanitizeSearch(useSearch({ strict: false }) as AdminUsersSearch)
  const allowedRoles = useMemo(() => getAssignableRoles(user), [user])
  const [filters, setFilters] = useState({
    role: search.role ?? '',
    status: search.status ?? '',
    username: search.username ?? '',
  })
  const [createOpen, setCreateOpen] = useState(false)
  const [createForm, setCreateForm] = useState<AdminUserCreateForm>(EMPTY_CREATE_FORM)
  const [resetTarget, setResetTarget] = useState<AdminUser | null>(null)
  const [resetForm, setResetForm] = useState<AdminPasswordResetForm>(EMPTY_RESET_FORM)
  const [roleTarget, setRoleTarget] = useState<AdminUser | null>(null)
  const [roleForm, setRoleForm] = useState<AdminRoleChangeForm>({ role: 'standard' })
  const [statusTarget, setStatusTarget] = useState<AdminUser | null>(null)
  const [statusAction, setStatusAction] = useState<'disable' | 'enable' | null>(null)
  const [notice, setNotice] = useState<{ text: string; type: 'error' | 'success' } | null>(null)

  useEffect(() => {
    setFilters({
      role: search.role ?? '',
      status: search.status ?? '',
      username: search.username ?? '',
    })
  }, [search.role, search.status, search.username])

  const usersQuery = useAdminUsers(search)
  const createMutation = useCreateAdminUser()
  const updateMutation = useUpdateAdminUser()
  const resetMutation = useResetAdminUserPassword()

  const isMutating = createMutation.isPending || updateMutation.isPending || resetMutation.isPending

  const replaceSearch = (next: Partial<AdminUsersSearch>) => {
    void navigate({
      to: '/admin/users',
      search: sanitizeSearch({
        ...search,
        ...next,
      }),
    })
  }

  const applyFilters = () => {
    replaceSearch({
      page: 1,
      role: filters.role ? (filters.role as ManagedUserRole) : undefined,
      status: filters.status ? (filters.status as UserStatus) : undefined,
      username: filters.username.trim() || undefined,
    })
  }

  const resetFilters = () => {
    setFilters({ role: '', status: '', username: '' })
    replaceSearch({
      page: 1,
      role: undefined,
      status: undefined,
      username: undefined,
    })
  }

  const openCreate = () => {
    setCreateForm({ ...EMPTY_CREATE_FORM, role: allowedRoles[0] ?? 'standard' })
    setNotice(null)
    setCreateOpen(true)
  }

  const submitCreate = () => {
    const validation = validateCreateAdminUserForm(createForm, allowedRoles)
    if (!validation.valid || !validation.request) {
      setNotice({ type: 'error', text: validation.message ?? '表单校验失败' })
      return
    }

    createMutation.mutate(validation.request, {
      onSuccess: () => {
        setCreateOpen(false)
        setNotice({ type: 'success', text: '用户已创建，首次登录需要修改临时密码' })
      },
      onError: (error) => {
        setNotice({ type: 'error', text: formatUserManagementError(error, '创建用户失败') })
      },
    })
  }

  const openReset = (row: AdminUser) => {
    setResetTarget(row)
    setResetForm(EMPTY_RESET_FORM)
    setNotice(null)
  }

  const submitReset = () => {
    if (!resetTarget) return
    const validation = validatePasswordResetForm(resetForm)
    if (!validation.valid || !validation.request) {
      setNotice({ type: 'error', text: validation.message ?? '表单校验失败' })
      return
    }

    resetMutation.mutate(
      { userId: resetTarget.id, body: validation.request },
      {
        onSuccess: () => {
          setResetTarget(null)
          setNotice({ type: 'success', text: '临时密码已重置，目标用户下次登录需要改密' })
        },
        onError: (error) => {
          setNotice({ type: 'error', text: formatUserManagementError(error, '重置密码失败') })
        },
      },
    )
  }

  const openRoleChange = (row: AdminUser) => {
    setRoleTarget(row)
    setRoleForm({ role: primaryManagedRole(row) ?? 'standard' })
    setNotice(null)
  }

  const submitRoleChange = () => {
    if (!roleTarget) return
    const rowRoles = roleTarget.manageableRoles?.length ? roleTarget.manageableRoles : allowedRoles
    const validation = validateRoleChangeForm(
      roleForm,
      rowRoles.filter((role) => allowedRoles.includes(role)),
    )
    if (!validation.valid || !validation.request) {
      setNotice({ type: 'error', text: validation.message ?? '表单校验失败' })
      return
    }

    updateMutation.mutate(
      { userId: roleTarget.id, body: validation.request },
      {
        onSuccess: () => {
          setRoleTarget(null)
          setNotice({ type: 'success', text: '用户角色已更新' })
        },
        onError: (error) => {
          setNotice({ type: 'error', text: formatUserManagementError(error, '更新角色失败') })
        },
      },
    )
  }

  const openStatusConfirm = (row: AdminUser, action: 'disable' | 'enable') => {
    setStatusTarget(row)
    setStatusAction(action)
    setNotice(null)
  }

  const submitStatusChange = () => {
    if (!statusTarget || !statusAction) return
    const nextStatus = statusAction === 'disable' ? 'disabled' : 'active'

    updateMutation.mutate(
      { userId: statusTarget.id, body: { status: nextStatus } },
      {
        onSuccess: () => {
          setStatusTarget(null)
          setStatusAction(null)
          setNotice({
            type: 'success',
            text: nextStatus === 'disabled' ? '用户已禁用' : '用户已启用',
          })
        },
        onError: (error) => {
          setNotice({ type: 'error', text: formatUserManagementError(error, '更新状态失败') })
        },
      },
    )
  }

  if (!canManageAdminUsers(user)) {
    return (
      <StateBlock
        icon={ShieldAlert}
        title="没有用户管理权限"
        description="只有管理员或超级管理员可以访问用户管理。"
        variant="forbidden"
      />
    )
  }

  const page = usersQuery.data?.page
  const users = usersQuery.data?.items ?? []
  const totalPages = page ? Math.max(1, Math.ceil(page.total / page.pageSize)) : 1
  const currentUserId = user?.id

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">用户管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            管理可授权范围内的普通用户和管理员账号。
          </p>
        </div>
        <Button disabled={!allowedRoles.length} onClick={openCreate}>
          <Plus className="size-4" />
          新建用户
        </Button>
      </div>

      {notice && (
        <InlineNotice variant={notice.type === 'success' ? 'success' : 'error'}>
          {notice.text}
        </InlineNotice>
      )}

      <section className="rounded-lg border border-border bg-card p-4">
        <div className="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_160px_160px_auto_auto]">
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">用户名搜索</span>
            <Input
              placeholder="输入用户名"
              value={filters.username}
              onChange={(event) =>
                setFilters((prev) => ({ ...prev, username: event.target.value }))
              }
              onKeyDown={(event) => {
                if (event.key === 'Enter') applyFilters()
              }}
            />
          </label>
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">角色</span>
            <select
              className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm"
              value={filters.role}
              onChange={(event) => setFilters((prev) => ({ ...prev, role: event.target.value }))}
            >
              <option value="">全部角色</option>
              {allowedRoles.map((role) => (
                <option key={role} value={role}>
                  {roleLabel(role)}
                </option>
              ))}
            </select>
          </label>
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">状态</span>
            <select
              className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm"
              value={filters.status}
              onChange={(event) => setFilters((prev) => ({ ...prev, status: event.target.value }))}
            >
              <option value="">全部状态</option>
              {USER_STATUS_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {option.label}
                </option>
              ))}
            </select>
          </label>
          <div className="flex items-end">
            <Button className="w-full" onClick={applyFilters}>
              <Search className="size-4" />
              查询
            </Button>
          </div>
          <div className="flex items-end">
            <Button className="w-full" variant="outline" onClick={resetFilters}>
              重置
            </Button>
          </div>
        </div>
      </section>

      {usersQuery.isLoading ? (
        <TableSkeleton columns={8} rows={6} showToolbar={false} />
      ) : usersQuery.isError ? (
        <StateBlock
          action={
            <Button variant="outline" onClick={() => void usersQuery.refetch()}>
              <RefreshCw className="size-4" />
              重试
            </Button>
          }
          title="用户列表加载失败"
          description={formatUserManagementError(usersQuery.error, '用户列表加载失败')}
          variant="error"
        />
      ) : users.length === 0 ? (
        <StateBlock title="暂无可管理用户" description="当前筛选条件下没有用户。" variant="empty" />
      ) : (
        <section className="overflow-hidden rounded-lg border border-border bg-card">
          <div className="overflow-x-auto">
            <table className="min-w-[1180px] w-full text-left text-sm">
              <thead className="border-b border-border bg-muted/40 text-xs text-muted-foreground">
                <tr>
                  <th className="px-4 py-3 font-medium">用户</th>
                  <th className="px-4 py-3 font-medium">资料</th>
                  <th className="px-4 py-3 font-medium">状态</th>
                  <th className="px-4 py-3 font-medium">角色</th>
                  <th className="px-4 py-3 font-medium">权限</th>
                  <th className="px-4 py-3 font-medium">改密</th>
                  <th className="px-4 py-3 font-medium">时间</th>
                  <th className="px-4 py-3 text-right font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {users.map((row) => {
                  const self = isSelf(row, currentUserId)
                  const rowRoleOptions = row.manageableRoles?.length
                    ? row.manageableRoles.filter((role) => allowedRoles.includes(role))
                    : allowedRoles

                  return (
                    <tr key={row.id} className="align-top">
                      <td className="px-4 py-3">
                        <div className="font-medium text-foreground">{row.username}</div>
                        <div className="mt-1 max-w-52 truncate text-xs text-muted-foreground">
                          {row.id}
                        </div>
                        {self && (
                          <Badge className="mt-2" variant="outline">
                            当前账号
                          </Badge>
                        )}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        <div>{row.displayName || '-'}</div>
                        <div className="mt-1">{row.email || '-'}</div>
                        <div className="mt-1">{row.phone || '-'}</div>
                      </td>
                      <td className="px-4 py-3">
                        <Badge variant={statusBadgeVariant(row.status)}>
                          {statusLabel(row.status)}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex max-w-44 flex-wrap gap-1">
                          {row.roles.map((role) => (
                            <Badge key={role} variant="secondary">
                              {roleLabel(role)}
                            </Badge>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex max-h-16 max-w-56 flex-wrap gap-1 overflow-hidden">
                          {row.permissions.slice(0, 5).map((permission) => (
                            <Badge key={permission} variant="outline">
                              {permission}
                            </Badge>
                          ))}
                          {row.permissions.length > 5 && (
                            <Badge variant="outline">+{row.permissions.length - 5}</Badge>
                          )}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {row.mustChangePassword ? '需要' : '不需要'}
                      </td>
                      <td className="px-4 py-3 text-xs text-muted-foreground">
                        <div>创建：{formatDate(row.createdAt)}</div>
                        <div className="mt-1">更新：{formatDate(row.updatedAt)}</div>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex flex-wrap justify-end gap-1.5">
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={
                              !canRowChangeRole(row, allowedRoles, currentUserId) ||
                              !rowRoleOptions.length
                            }
                            onClick={() => openRoleChange(row)}
                          >
                            <Edit3 className="size-3.5" />
                            角色
                          </Button>
                          {row.status === 'disabled' ? (
                            <Button
                              size="sm"
                              variant="outline"
                              disabled={!canRowEnable(row, currentUserId)}
                              onClick={() => openStatusConfirm(row, 'enable')}
                            >
                              <UserCheck className="size-3.5" />
                              启用
                            </Button>
                          ) : (
                            <Button
                              size="sm"
                              variant="destructive"
                              disabled={!canRowDisable(row, currentUserId)}
                              onClick={() => openStatusConfirm(row, 'disable')}
                            >
                              <UserX className="size-3.5" />
                              禁用
                            </Button>
                          )}
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={!canRowReset(row, currentUserId)}
                            onClick={() => openReset(row)}
                          >
                            <RotateCcwKey className="size-3.5" />
                            重置
                          </Button>
                        </div>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
          {page && (
            <div className="flex flex-wrap items-center justify-between gap-3 border-t border-border px-4 py-3 text-sm text-muted-foreground">
              <span>
                第 {page.page} / {totalPages} 页，共 {page.total} 条
              </span>
              <div className="flex items-center gap-2">
                <select
                  aria-label="每页条数"
                  className="h-8 rounded-lg border border-input bg-background px-2 text-sm"
                  value={search.pageSize}
                  onChange={(event) =>
                    replaceSearch({ page: 1, pageSize: Number(event.target.value) })
                  }
                >
                  {[10, 20, 50].map((size) => (
                    <option key={size} value={size}>
                      {size} / 页
                    </option>
                  ))}
                </select>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page.page <= 1}
                  onClick={() => replaceSearch({ page: page.page - 1 })}
                >
                  上一页
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  disabled={page.page >= totalPages}
                  onClick={() => replaceSearch({ page: page.page + 1 })}
                >
                  下一页
                </Button>
              </div>
            </div>
          )}
        </section>
      )}

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent className="sm:max-w-xl">
          <DialogHeader>
            <DialogTitle>新建用户</DialogTitle>
            <DialogDescription>
              管理员手动输入临时密码。创建后目标用户首次登录需要修改密码。
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 sm:grid-cols-2">
            <label className="space-y-1.5 text-sm">
              <span className="text-muted-foreground">用户名</span>
              <Input
                value={createForm.username}
                disabled={isMutating}
                onChange={(event) =>
                  setCreateForm((prev) => ({ ...prev, username: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1.5 text-sm">
              <span className="text-muted-foreground">角色</span>
              <select
                className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm"
                value={createForm.role}
                disabled={isMutating}
                onChange={(event) =>
                  setCreateForm((prev) => ({
                    ...prev,
                    role: event.target.value as ManagedUserRole,
                  }))
                }
              >
                {allowedRoles.map((role) => (
                  <option key={role} value={role}>
                    {roleLabel(role)}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-1.5 text-sm sm:col-span-2">
              <span className="text-muted-foreground">临时密码</span>
              <Input
                autoComplete="new-password"
                type="password"
                value={createForm.temporaryPassword}
                disabled={isMutating}
                onChange={(event) =>
                  setCreateForm((prev) => ({
                    ...prev,
                    temporaryPassword: event.target.value,
                  }))
                }
              />
            </label>
            <label className="space-y-1.5 text-sm">
              <span className="text-muted-foreground">显示名</span>
              <Input
                value={createForm.displayName}
                disabled={isMutating}
                onChange={(event) =>
                  setCreateForm((prev) => ({ ...prev, displayName: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1.5 text-sm">
              <span className="text-muted-foreground">邮箱</span>
              <Input
                type="email"
                value={createForm.email}
                disabled={isMutating}
                placeholder="可不填写"
                onChange={(event) =>
                  setCreateForm((prev) => ({ ...prev, email: event.target.value }))
                }
              />
            </label>
            <label className="space-y-1.5 text-sm sm:col-span-2">
              <span className="text-muted-foreground">电话</span>
              <Input
                value={createForm.phone}
                disabled={isMutating}
                placeholder="可不填写"
                onChange={(event) =>
                  setCreateForm((prev) => ({ ...prev, phone: event.target.value }))
                }
              />
            </label>
          </div>
          <DialogFooter>
            <Button variant="outline" disabled={isMutating} onClick={() => setCreateOpen(false)}>
              取消
            </Button>
            <Button disabled={isMutating || !allowedRoles.length} onClick={submitCreate}>
              {createMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              创建
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(resetTarget)} onOpenChange={(open) => !open && setResetTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>重置临时密码</DialogTitle>
            <DialogDescription>
              为 {resetTarget?.username} 设置新的临时密码。目标用户下次登录必须改密。
            </DialogDescription>
          </DialogHeader>
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">新临时密码</span>
            <Input
              autoComplete="new-password"
              type="password"
              value={resetForm.temporaryPassword}
              disabled={isMutating}
              onChange={(event) => setResetForm({ temporaryPassword: event.target.value })}
            />
          </label>
          <DialogFooter>
            <Button variant="outline" disabled={isMutating} onClick={() => setResetTarget(null)}>
              取消
            </Button>
            <Button disabled={isMutating} onClick={submitReset}>
              {resetMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              重置密码
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={Boolean(roleTarget)} onOpenChange={(open) => !open && setRoleTarget(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>替换用户角色</DialogTitle>
            <DialogDescription>
              为 {roleTarget?.username} 设置一个新的托管角色。公开 UI 不支持 super_admin。
            </DialogDescription>
          </DialogHeader>
          <label className="space-y-1.5 text-sm">
            <span className="text-muted-foreground">目标角色</span>
            <select
              className="h-8 w-full rounded-lg border border-input bg-background px-2.5 text-sm"
              value={roleForm.role}
              disabled={isMutating}
              onChange={(event) => setRoleForm({ role: event.target.value as ManagedUserRole })}
            >
              {(roleTarget?.manageableRoles?.length
                ? roleTarget.manageableRoles.filter((role) => allowedRoles.includes(role))
                : allowedRoles
              ).map((role) => (
                <option key={role} value={role}>
                  {roleLabel(role)}
                </option>
              ))}
            </select>
          </label>
          <DialogFooter>
            <Button variant="outline" disabled={isMutating} onClick={() => setRoleTarget(null)}>
              取消
            </Button>
            <Button disabled={isMutating} onClick={submitRoleChange}>
              {updateMutation.isPending ? <Loader2 className="size-4 animate-spin" /> : null}
              保存角色
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={Boolean(statusTarget && statusAction)}
        onOpenChange={(open) => {
          if (!open) {
            setStatusTarget(null)
            setStatusAction(null)
          }
        }}
        title={statusAction === 'disable' ? '禁用用户' : '启用用户'}
        description={
          statusAction === 'disable'
            ? `确认禁用 ${statusTarget?.username ?? '该用户'}？`
            : `确认重新启用 ${statusTarget?.username ?? '该用户'}？`
        }
        confirmLabel={statusAction === 'disable' ? '确认禁用' : '确认启用'}
        pending={updateMutation.isPending}
        variant={statusAction === 'disable' ? 'destructive' : 'default'}
        onConfirm={submitStatusChange}
      />
    </div>
  )
}
