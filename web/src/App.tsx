import {
  createRouter,
  createRootRoute,
  createRoute,
  RouterProvider,
  Navigate,
} from '@tanstack/react-router'
import { ThemeProvider } from 'next-themes'
import { Toaster } from '@/components/ui/sonner'
import { AppShell } from '@/components/layout/AppShell'
import { ObjectListView } from '@/components/objects/ObjectListView'
import { PipelineBoard } from '@/components/views/PipelineBoard'
import { PropertyManager } from '@/components/views/PropertyManager'
import { PipelineSettings } from '@/components/views/PipelineSettings'
import { AdminControls } from '@/components/views/AdminControls'
import { RecordDetailPage } from '@/components/views/RecordDetailPage'
import { DashboardPage } from '@/components/views/DashboardPage'

// Root route with layout
const rootRoute = createRootRoute({
  component: AppShell,
})

// Index redirect: /_ui/ → /_ui/dashboard
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: () => <Navigate to="/dashboard" />,
})

// Dashboard: /_ui/dashboard
const dashboardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/dashboard',
  component: DashboardPage,
})

// Object list: /_ui/:objectType
const objectListRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/$objectType',
  component: function ObjectListPage() {
    const { objectType } = objectListRoute.useParams()
    return <ObjectListView objectType={objectType} />
  },
})

// Pipeline board: /_ui/:objectType/board
const pipelineBoardRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/$objectType/board',
  component: PipelineBoard,
})

// Settings: properties
const propertiesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/properties',
  component: PropertyManager,
})

// Settings: pipelines
const pipelinesRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/settings/pipelines',
  component: PipelineSettings,
})

// Admin
const adminRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/admin',
  component: AdminControls,
})

// Object detail: /_ui/:objectType/:objectId
const objectDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/$objectType/$objectId',
  component: function ObjectDetailPageWrapper() {
    const { objectType, objectId } = objectDetailRoute.useParams()
    return <RecordDetailPage objectType={objectType} objectId={objectId} />
  },
})

// Build route tree
const routeTree = rootRoute.addChildren([
  indexRoute,
  dashboardRoute,
  propertiesRoute,
  pipelinesRoute,
  adminRoute,
  pipelineBoardRoute,
  objectDetailRoute,
  objectListRoute,
])

// Create router with base path
const router = createRouter({
  routeTree,
  basepath: '/_ui',
})

// Register the router for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

function App() {
  return (
    <ThemeProvider attribute="class" defaultTheme="light" disableTransitionOnChange>
      <RouterProvider router={router} />
      <Toaster position="bottom-right" />
    </ThemeProvider>
  )
}

export default App
