import { Link, useRouterState } from '@tanstack/react-router'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useState } from 'react'

import type { AdminMenuItem } from '@/lib/types'
import { cn } from '@/lib/utils'

const menuItems: AdminMenuItem[] = [
  {
    key: 'system',
    label: '系统管理',
    children: [
      { key: 'users', label: '用户管理', path: '/admin/users' },
      { key: 'roles', label: '角色管理', path: '/admin/roles' },
      { key: 'styles', label: '样式管理', path: '/admin/styles' },
      { key: 'report-categories', label: '报告类别', path: '/admin/report-categories' },
      { key: 'files', label: '文件管理', path: '/admin/files' },
    ],
  },
  {
    key: 'stats',
    label: '统计概览',
    path: '/admin/stats',
  },
  {
    key: 'templates',
    label: '模板管理',
    path: '/admin/templates',
  },
  {
    key: 'materials',
    label: '材料管理',
    path: '/admin/materials',
  },
  {
    key: 'prompts',
    label: '提示词管理',
    path: '/admin/prompts',
  },
  {
    key: 'rag',
    label: 'RAG 知识库',
    children: [
      { key: 'knowledge', label: '知识管理', path: '/admin/knowledge' },
      { key: 'knowledge-config', label: '知识配置', path: '/admin/knowledge-config' },
      { key: 'knowledge-experience', label: '知识体验', path: '/admin/knowledge-experience' },
    ],
  },
  {
    key: 'settings',
    label: '系统设置',
    path: '/admin/settings',
  },
]

export function AdminSidebar() {
  const routerState = useRouterState()
  const pathname = routerState.location.pathname
  const [expanded, setExpanded] = useState<Set<string>>(new Set(['system', 'rag']))

  const toggle = (key: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(key)) {
        next.delete(key)
      } else {
        next.add(key)
      }
      return next
    })
  }

  const isActive = (path?: string): boolean => {
    if (!path) return false
    return pathname === path || pathname.startsWith(`${path}/`)
  }

  return (
    <aside className="flex w-56 flex-shrink-0 flex-col border-r border-border bg-sidebar">
      {/* Title */}
      <h2 className="border-b border-border px-4 py-3 text-sm font-semibold text-sidebar-foreground">
        管理面板
      </h2>

      {/* Navigation */}
      <nav className="flex flex-1 flex-col gap-0.5 overflow-auto py-1">
        {menuItems.map((item) => {
          const hasChildren = item.children && item.children.length > 0

          if (hasChildren) {
            const open = expanded.has(item.key)
            return (
              <div key={item.key}>
                {/* Group header */}
                <button
                  type="button"
                  className={cn(
                    'flex w-full items-center gap-1.5 px-4 py-2 text-left text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-accent',
                  )}
                  onClick={() => toggle(item.key)}
                >
                  {open ? (
                    <ChevronDown
                      aria-hidden="true"
                      size={12}
                      className="shrink-0 text-muted-foreground"
                    />
                  ) : (
                    <ChevronRight
                      aria-hidden="true"
                      size={12}
                      className="shrink-0 text-muted-foreground"
                    />
                  )}
                  <span>{item.label}</span>
                </button>

                {/* Children */}
                {open && (
                  <div className="bg-sidebar-accent/40 py-0.5">
                    {item.children!.map((child) => (
                      <Link
                        key={child.key}
                        to={child.path!}
                        className={cn(
                          'block px-4 py-1.5 pl-10 text-sm text-muted-foreground transition-colors hover:bg-sidebar-accent hover:text-sidebar-foreground',
                          isActive(child.path) &&
                            'bg-sidebar-accent text-sidebar-accent-foreground font-medium',
                        )}
                      >
                        {child.label}
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            )
          }

          // Single item (no children)
          return (
            <Link
              key={item.key}
              to={item.path!}
              className={cn(
                'flex w-full items-center px-4 py-2 text-sm font-medium text-sidebar-foreground transition-colors hover:bg-sidebar-accent',
                isActive(item.path) && 'bg-sidebar-accent text-sidebar-accent-foreground',
              )}
            >
              {item.label}
            </Link>
          )
        })}
      </nav>
    </aside>
  )
}
