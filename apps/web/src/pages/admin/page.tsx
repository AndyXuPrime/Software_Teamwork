import { Link, Outlet } from '@tanstack/react-router'

import { AdminSidebar } from '@/components/admin/admin-sidebar'

export function AdminPage() {
  return (
    <div className="flex h-full">
      {/* Admin sidebar */}
      <AdminSidebar />

      {/* Content area */}
      <main className="flex min-w-0 flex-1 flex-col overflow-auto">
        {/* Back link */}
        <div className="border-b border-border px-6 py-3">
          <Link
            to="/"
            className="inline-block text-sm text-muted-foreground transition-colors hover:text-primary"
          >
            &larr; 返回首页
          </Link>
        </div>

        {/* Page content */}
        <div className="flex-1 p-6">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
