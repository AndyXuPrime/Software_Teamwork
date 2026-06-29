import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
  redirect,
  useNavigate,
  useParams,
} from '@tanstack/react-router'

import { AppLayout } from '@/layouts/app-layout'
import { KnowledgeConfig } from '@/pages/admin/knowledge-config'
import { KnowledgeManagement } from '@/pages/admin/knowledge-management'
import { AdminPage } from '@/pages/admin/page'
import { QARetrievalTestPage } from '@/pages/admin/qa-retrieval-test'
import { QASettings } from '@/pages/admin/qa-settings'
import { StatsOverviewPage } from '@/pages/admin/stats-overview'
import { StyleManagement } from '@/pages/admin/style-management'
import { SystemSettings } from '@/pages/admin/system-settings'
import { KnowledgeChunksPage } from '@/pages/knowledge/chunks/page'
import { KnowledgeDocumentsPage } from '@/pages/knowledge/documents/page'
import { KnowledgeSearchPage } from '@/pages/knowledge/search/page'
import { ChatPage } from '@/pages/qa/chat/page'
import { ReportGeneratePage } from '@/pages/reports/generate/page'
import { ReportRecordsPage } from '@/pages/reports/records/page'
import { ReportTemplatesPage } from '@/pages/reports/templates/page'
import { useAuthStore } from '@/stores/auth-store'

async function restoreAuthForRoute() {
  const store = useAuthStore.getState()

  if (store.status === 'idle' || (store.accessToken && !store.user)) {
    await store.restoreSession()
  }

  return useAuthStore.getState()
}

function requireAuth(requirement?: PermissionRequirement) {
  return async () => {
    const store = await restoreAuthForRoute()

    if (store.status === 'anonymous') {
      throw redirect({ to: '/login' })
    }

    if (store.status === 'error') {
      return
    }

    if (!canAccess(store.user, requirement)) {
      throw redirect({ to: '/forbidden' })
    }
  }
}

async function redirectToReportHome() {
  const store = await restoreAuthForRoute()

  if (canAccess(store.user, reportWriteAccess)) {
    throw redirect({ to: '/reports/generate' })
  }

  if (canAccess(store.user, reportAccess)) {
    throw redirect({ to: '/reports/records' })
  }

  throw redirect({ to: '/forbidden' })
}

async function redirectToAppHome() {
  const store = await restoreAuthForRoute()

  if (canAccess(store.user, qaAccess)) {
    throw redirect({ to: '/chat' })
  }

  if (canAccess(store.user, reportWriteAccess)) {
    throw redirect({ to: '/reports/generate' })
  }

  if (canAccess(store.user, reportAccess)) {
    throw redirect({ to: '/reports/records' })
  }

  if (canAccess(store.user, adminAccess)) {
    throw redirect({ to: '/admin' })
  }

  throw redirect({ to: '/forbidden' })
}

async function redirectToAdminHome() {
  const store = await restoreAuthForRoute()

  if (canAccess(store.user, systemAdminAccess)) {
    throw redirect({ to: '/admin/stats' })
  }

  if (canAccess(store.user, reportAccess)) {
    throw redirect({ to: '/admin/reports/records' })
  }

  if (canAccess(store.user, knowledgeWriteAccess)) {
    throw redirect({ to: '/admin/knowledge' })
  }

  if (canAccess(store.user, knowledgeAccess)) {
    throw redirect({ to: '/admin/knowledge-config' })
  }

  if (canAccess(store.user, qaAdminAccess)) {
    throw redirect({ to: '/admin/prompts' })
  }

  throw redirect({ to: '/forbidden' })
}

const qaAccess: PermissionRequirement = { any: ['qa:use'] }
const qaAdminAccess: PermissionRequirement = {
  any: ['admin:model-profile:write', 'admin:parser-config:write', 'system:admin'],
}
const reportAccess: PermissionRequirement = {
  any: ['report:read', 'report:write', 'reports:write'],
}
const reportWriteAccess: PermissionRequirement = { any: ['report:write', 'reports:write'] }
const knowledgeAccess: PermissionRequirement = {
  any: ['knowledge:read', 'knowledge:write', 'document:upload'],
}
const knowledgeWriteAccess: PermissionRequirement = { any: ['knowledge:write'] }
const systemAdminAccess: PermissionRequirement = { any: ['system:admin'] }
const adminAccess: PermissionRequirement = {
  any: [
    'system:admin',
    'report:read',
    'report:write',
    'reports:write',
    'knowledge:read',
    'knowledge:write',
    'document:upload',
    'admin:model-profile:write',
    'admin:parser-config:write',
  ],
}

const rootRoute = createRootRoute({
  component: Outlet,
})

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: 'login',
  beforeLoad: async () => {
    const store = await restoreAuthForRoute()
    if (store.status === 'authenticated') {
      await redirectToAppHome()
    }
  },
  component: LoginPage,
})

const authenticatedRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'authenticated',
  beforeLoad: requireAuth(),
  component: () => (
    <AppLayout>
      <Outlet />
    </AppLayout>
  ),
})

// ── Child routes ────────────────────────────────────────────

const indexRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: '/',
  beforeLoad: redirectToAppHome,
})

const forbiddenRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'forbidden',
  component: ForbiddenPage,
})

const chatRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'chat',
  beforeLoad: requireAuth(qaAccess),
  component: ChatPage,
})

const reportsRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'reports',
  beforeLoad: requireAuth(reportAccess),
  component: Outlet,
})

const reportsIndexRoute = createRoute({
  getParentRoute: () => reportsRoute,
  path: '/',
  beforeLoad: redirectToReportHome,
})

const reportGenerateRoute = createRoute({
  getParentRoute: () => reportsRoute,
  path: 'generate',
  beforeLoad: requireAuth(reportWriteAccess),
  component: ReportGeneratePage,
})

const reportRecordsRoute = createRoute({
  getParentRoute: () => reportsRoute,
  path: 'records',
  component: ReportRecordsPage,
})

const reportTemplatesRoute = createRoute({
  getParentRoute: () => reportsRoute,
  path: 'templates',
  beforeLoad: requireAuth(reportWriteAccess),
  component: ReportTemplatesPage,
})

const adminRoute = createRoute({
  getParentRoute: () => authenticatedRoute,
  path: 'admin',
  beforeLoad: requireAuth(adminAccess),
  component: AdminPage,
})

const adminIndexRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: '/',
  beforeLoad: redirectToAdminHome,
})

const adminStylesRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'styles',
  beforeLoad: requireAuth(systemAdminAccess),
  component: StyleManagement,
})

const adminKnowledgeRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'knowledge',
  beforeLoad: requireAuth(knowledgeWriteAccess),
  component: KnowledgeManagement,
})

const adminKnowledgeConfigRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'knowledge-config',
  beforeLoad: requireAuth(knowledgeAccess),
  component: KnowledgeConfig,
})

const adminQASettingsRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'qa-settings',
  beforeLoad: requireAuth(qaAdminAccess),
  component: QASettings,
})

const adminQARetrievalTestRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'qa-retrieval-test',
  beforeLoad: requireAuth(qaAdminAccess),
  component: QARetrievalTestPage,
})

const adminSettingsRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'settings',
  beforeLoad: requireAuth(systemAdminAccess),
  component: SystemSettings,
})

const adminStatsRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'stats',
  beforeLoad: requireAuth(systemAdminAccess),
  component: StatsOverviewPage,
})

const adminReportRecordsRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'reports/records',
  beforeLoad: requireAuth(reportAccess),
  component: ReportRecordsPage,
})

const adminReportTemplatesRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'reports/templates',
  beforeLoad: requireAuth(reportWriteAccess),
  component: ReportTemplatesPage,
})

// ── Knowledge document management routes ─────────────────────

function KnowledgeDocumentsRoute() {
  const navigate = useNavigate()
  const searchParams = new URLSearchParams(window.location.search)
  const knowledgeBaseId = searchParams.get('knowledgeBaseId') ?? ''

  const handleNavigateChunks = (documentId: string) => {
    void navigate({
      to: '/admin/knowledge/chunks/$documentId',
      params: { documentId },
    })
  }

  return (
    <KnowledgeDocumentsPage
      knowledgeBaseId={knowledgeBaseId || undefined}
      onNavigateChunks={handleNavigateChunks}
    />
  )
}

function KnowledgeChunksRoute() {
  const params = useParams({ strict: false }) as Record<string, string>
  const documentId = params.documentId ?? ''

  if (!documentId) {
    return (
      <div className="rounded-lg border border-destructive/50 bg-destructive/10 p-6 text-center">
        <p className="text-sm text-destructive">缺少文档 ID 参数</p>
      </div>
    )
  }

  return (
    <KnowledgeChunksPage
      documentId={documentId}
      onNavigateBack={() => {
        window.history.back()
      }}
    />
  )
}

const adminKnowledgeDocumentsRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'knowledge/documents',
  component: KnowledgeDocumentsRoute,
})

const adminKnowledgeChunksRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'knowledge/chunks/$documentId',
  component: KnowledgeChunksRoute,
})

const adminKnowledgeSearchRoute = createRoute({
  getParentRoute: () => adminRoute,
  path: 'knowledge/search',
  component: KnowledgeSearchPage,
})

// ── Route tree ──────────────────────────────────────────────
const routeTree = rootRoute.addChildren([
  indexRoute,
  chatRoute,
  reportsRoute.addChildren([
    reportsIndexRoute,
    reportGenerateRoute,
    reportRecordsRoute,
    reportTemplatesRoute,
  ]),
  adminRoute.addChildren([
    adminIndexRoute,
    adminStylesRoute,
    adminKnowledgeRoute,
    adminKnowledgeConfigRoute,
    adminKnowledgeDocumentsRoute,
    adminKnowledgeChunksRoute,
    adminKnowledgeSearchRoute,
    adminQASettingsRoute,
    adminQARetrievalTestRoute,
    adminSettingsRoute,
    adminStatsRoute,
    adminReportRecordsRoute,
    adminReportTemplatesRoute,
  ]),
])

export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
