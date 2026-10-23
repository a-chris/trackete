import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { errorMessage, isUnauthorized } from '../lib/api'
import { useMe } from '../lib/queries'
import { ErrorState } from './ui/ErrorState'
import { FullPageSpinner } from './ui/Spinner'

export function RequireAuth() {
  const { data, isLoading, isError, error, refetch, isFetching } = useMe()
  const location = useLocation()

  if (isLoading) return <FullPageSpinner />
  if (isError) {
    if (isUnauthorized(error)) {
      return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
    }
    return (
      <div className="grid min-h-screen place-items-center px-4">
        <div className="w-full max-w-md">
          <ErrorState
            title="Can't reach the server"
            description={errorMessage(error)}
            onRetry={() => void refetch()}
            retrying={isFetching}
          />
        </div>
      </div>
    )
  }
  if (!data) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  }
  return <Outlet />
}
