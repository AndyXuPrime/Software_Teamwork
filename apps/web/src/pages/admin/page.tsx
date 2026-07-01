import { Outlet } from '@tanstack/react-router'

import { AdminSidebar } from '@/components/admin/admin-sidebar'

export function AdminPage() {
  return (
    <div className="flex h-full">
      {/* Admin sidebar */}
      <AdminSidebar />

      {/* Content area */}
      <main className="flex min-w-0 flex-1 flex-col overflow-auto p-6">
        <Outlet />
      </main>
    </div>
  )
}
