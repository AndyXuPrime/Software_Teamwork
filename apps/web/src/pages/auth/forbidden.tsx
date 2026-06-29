import { Link } from '@tanstack/react-router'
import { ShieldAlert } from 'lucide-react'

import { Button } from '@/components/ui/button'

export function ForbiddenPage() {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <section className="w-full max-w-md rounded-lg border border-border bg-card p-6 text-center shadow-sm">
        <div className="mx-auto mb-4 flex size-10 items-center justify-center rounded-md bg-destructive/10 text-destructive">
          <ShieldAlert className="size-5" />
        </div>
        <h1 className="text-lg font-semibold">权限不足</h1>
        <p className="mt-2 text-sm text-muted-foreground">
          当前账号没有访问该页面所需的角色或权限，请联系管理员调整授权。
        </p>
        <Link to="/">
          <Button className="mt-5" variant="outline">
            返回首页
          </Button>
        </Link>
      </section>
    </div>
  )
}
