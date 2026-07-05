import { useQuery } from '@tanstack/react-query'

import { getAdminOverview } from './admin-dashboard.api'

export const adminDashboardKeys = {
  all: ['admin-dashboard'] as const,
  overview: () => [...adminDashboardKeys.all, 'overview'] as const,
}

export function useAdminOverviewQuery() {
  return useQuery({
    queryKey: adminDashboardKeys.overview(),
    queryFn: getAdminOverview,
  })
}
