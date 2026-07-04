import { Outlet, useRouterState } from '@tanstack/react-router'

import { AdminSidebar } from '@/components/admin/admin-sidebar'

export function AdminPage() {
  const pathname = useRouterState({ select: (state) => state.location.pathname })

  return (
    <div className="flex h-full">
      {/* Admin sidebar */}
      <AdminSidebar />

      {/* Content area */}
      <main
        key={pathname}
        className="page-enter-right flex min-w-0 flex-1 flex-col overflow-auto p-6"
      >
        <Outlet />
      </main>
    </div>
  )
}
