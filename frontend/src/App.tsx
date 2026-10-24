import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/AppShell'
import { RequireAuth } from './components/RequireAuth'
import { FullPageSpinner } from './components/ui/Spinner'
import { LoginPage, RegisterPage } from './pages/AuthPages'
import { InvitePage } from './pages/InvitePage'
import { SettingsPage } from './pages/SettingsPage'
import { TodayPage } from './pages/TodayPage'
import { TrackersPage } from './pages/TrackersPage'

// Chart-heavy pages are split out so the initial bundle stays small.
const DashboardPage = lazy(() =>
  import('./pages/DashboardPage').then((module) => ({ default: module.DashboardPage })),
)
const TrackerDetailPage = lazy(() =>
  import('./pages/TrackerDetailPage').then((module) => ({ default: module.TrackerDetailPage })),
)

export function App() {
  return (
    <Suspense fallback={<FullPageSpinner />}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        {/* Public: the signed token is the credential, so no session is needed. */}
        <Route path="/invite/:token" element={<InvitePage />} />

        <Route element={<RequireAuth />}>
          <Route element={<AppShell />}>
            <Route index element={<TodayPage />} />
            <Route path="dashboard" element={<DashboardPage />} />
            <Route path="dashboard/:dashboardId" element={<DashboardPage />} />
            <Route path="trackers" element={<TrackersPage />} />
            <Route path="trackers/:trackerId" element={<TrackerDetailPage />} />
            <Route path="settings" element={<SettingsPage />} />
          </Route>
        </Route>

        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}
