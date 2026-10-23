import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useMe } from '../lib/queries'
import { FullPageSpinner } from './ui/Spinner'

export function RequireAuth() {
  const { data, isLoading, isError } = useMe()
  const location = useLocation()

  if (isLoading) return <FullPageSpinner />
  if (isError || !data) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  }
  return <Outlet />
}
