import { gatewayRequest } from '@/api/client'

import type { AdminOverview } from './admin-dashboard.types'

export function getAdminOverview(): Promise<AdminOverview> {
  return gatewayRequest<AdminOverview>('/admin/overview')
}
