import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, isUnauthorized } from './api'
import type { ChartData } from './types'

// A rejected session (401) is final: retrying only delays the redirect. Network
// blips and 5xx responses are worth one more attempt.
function retryUnlessUnauthorized(failureCount: number, error: unknown): boolean {
  return !isUnauthorized(error) && failureCount < 2
}

export function useMe() {
  return useQuery({
    queryKey: ['me'],
    queryFn: api.me,
    retry: retryUnlessUnauthorized,
    staleTime: 5 * 60_000,
  })
}

export function useMeta() {
  return useQuery({ queryKey: ['meta'], queryFn: api.meta, staleTime: 60 * 60_000 })
}

export function useTrackers(includeArchived = false) {
  return useQuery({
    queryKey: ['trackers', includeArchived],
    queryFn: () => api.listTrackers(includeArchived),
  })
}

export function useOverview(today: string) {
  return useQuery({
    queryKey: ['overview', today],
    queryFn: () => api.overview(today),
  })
}

export function useDashboards() {
  return useQuery({ queryKey: ['dashboards'], queryFn: api.dashboards })
}

export function useDashboardData(dashboardId: string | undefined, today: string) {
  return useQuery({
    queryKey: ['dashboard-data', dashboardId, today],
    queryFn: () => api.dashboardData(dashboardId!, today),
    enabled: Boolean(dashboardId),
  })
}

interface TrackerDataArgs {
  trackerId: string
  from: string
  to: string
  chart: string
  granularity?: string
  aggregate?: string
  metric?: string
  enabled?: boolean
}

export function useTrackerData(args: TrackerDataArgs) {
  return useQuery<ChartData>({
    queryKey: [
      'tracker-data',
      args.trackerId,
      args.from,
      args.to,
      args.chart,
      args.granularity,
      args.aggregate,
      args.metric,
    ],
    queryFn: () =>
      api.trackerData(args.trackerId, {
        from: args.from,
        to: args.to,
        chart: args.chart,
        granularity: args.granularity,
        aggregate: args.aggregate,
        metric: args.metric,
      }),
    enabled: args.enabled ?? true,
  })
}

export function useEntryMutations() {
  const queryClient = useQueryClient()
  const invalidate = () => {
    void queryClient.invalidateQueries()
  }

  const save = useMutation({
    mutationFn: (args: { trackerId: string; day: string; value: number; note?: string }) =>
      api.saveEntry(args.trackerId, args.day, args.value, args.note ?? ''),
    onSuccess: invalidate,
  })

  const remove = useMutation({
    mutationFn: (args: { trackerId: string; day: string }) => api.deleteEntry(args.trackerId, args.day),
    onSuccess: invalidate,
  })

  return { save, remove }
}

export function useTrackerMutations() {
  const queryClient = useQueryClient()
  const invalidate = () => {
    void queryClient.invalidateQueries()
  }
  return {
    create: useMutation({ mutationFn: api.createTracker, onSuccess: invalidate }),
    update: useMutation({
      mutationFn: (args: { id: string; input: Parameters<typeof api.updateTracker>[1] }) =>
        api.updateTracker(args.id, args.input),
      onSuccess: invalidate,
    }),
    remove: useMutation({ mutationFn: api.deleteTracker, onSuccess: invalidate }),
  }
}
