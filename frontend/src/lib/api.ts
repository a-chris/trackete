import type {
  ChartData,
  Dashboard,
  Entry,
  Meta,
  OverviewResponse,
  Tracker,
  TrackerInput,
  User,
  Widget,
  WidgetInput,
  WidgetLayout,
} from './types'

export class ApiError extends Error {
  status: number

  constructor(message: string, status: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      ...(init?.headers ?? {}),
    },
  })

  const text = await response.text()
  let payload: unknown = null
  if (text) {
    try {
      payload = JSON.parse(text)
    } catch {
      payload = null
    }
  }

  if (!response.ok) {
    const message =
      payload && typeof payload === 'object' && 'error' in payload
        ? String((payload as { error: unknown }).error)
        : `Request failed (${response.status})`
    throw new ApiError(message, response.status)
  }

  return payload as T
}

function query(params: Record<string, string | undefined>): string {
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') search.set(key, value)
  }
  const text = search.toString()
  return text ? `?${text}` : ''
}

const json = (body: unknown): RequestInit => ({ body: JSON.stringify(body) })

export const api = {
  meta: () => request<Meta>('/api/meta'),

  me: () => request<{ user: User }>('/api/auth/me').then((r) => r.user),
  login: (email: string, password: string) =>
    request<{ user: User }>('/api/auth/login', { method: 'POST', ...json({ email, password }) }).then(
      (r) => r.user,
    ),
  register: (email: string, password: string, name: string) =>
    request<{ user: User }>('/api/auth/register', {
      method: 'POST',
      ...json({ email, password, name }),
    }).then((r) => r.user),
  logout: () => request<{ ok: true }>('/api/auth/logout', { method: 'POST' }),
  updateMe: (name: string) =>
    request<{ user: User }>('/api/auth/me', { method: 'PATCH', ...json({ name }) }).then((r) => r.user),
  changePassword: (currentPassword: string, newPassword: string) =>
    request<{ ok: true }>('/api/auth/password', {
      method: 'POST',
      ...json({ currentPassword, newPassword }),
    }),

  overview: (today: string, weekStart?: string, monthStart?: string) =>
    request<OverviewResponse>(`/api/overview${query({ today, weekStart, monthStart })}`),

  listTrackers: (includeArchived = false) =>
    request<{ trackers: Tracker[] }>(`/api/trackers${query({ includeArchived: String(includeArchived) })}`).then(
      (r) => r.trackers,
    ),
  createTracker: (input: TrackerInput) =>
    request<Tracker>('/api/trackers', { method: 'POST', ...json(input) }),
  updateTracker: (id: string, input: TrackerInput) =>
    request<Tracker>(`/api/trackers/${id}`, { method: 'PUT', ...json(input) }),
  deleteTracker: (id: string) => request<{ ok: true }>(`/api/trackers/${id}`, { method: 'DELETE' }),

  entries: (trackerId: string, from: string, to: string) =>
    request<{ entries: Entry[] }>(`/api/trackers/${trackerId}/entries${query({ from, to })}`).then(
      (r) => r.entries,
    ),
  saveEntry: (trackerId: string, day: string, value: number, note = '') =>
    request<Entry>(`/api/trackers/${trackerId}/entries/${day}`, {
      method: 'PUT',
      ...json({ value, note }),
    }),
  deleteEntry: (trackerId: string, day: string) =>
    request<{ ok: true }>(`/api/trackers/${trackerId}/entries/${day}`, { method: 'DELETE' }),

  trackerData: (
    trackerId: string,
    params: {
      from?: string
      to: string
      range?: string
      chart: string
      granularity?: string
      aggregate?: string
      metric?: string
    },
  ) => request<ChartData>(`/api/trackers/${trackerId}/data${query(params)}`),

  dashboards: () =>
    request<{ dashboards: Dashboard[] }>('/api/dashboards').then((r) => r.dashboards),
  createDashboard: (name: string) =>
    request<Dashboard>('/api/dashboards', { method: 'POST', ...json({ name }) }),
  renameDashboard: (id: string, name: string) =>
    request<Dashboard>(`/api/dashboards/${id}`, { method: 'PATCH', ...json({ name }) }),
  deleteDashboard: (id: string) =>
    request<{ ok: true }>(`/api/dashboards/${id}`, { method: 'DELETE' }),

  dashboardData: (id: string, to: string) =>
    request<{ to: string; widgets: Record<string, ChartData> }>(
      `/api/dashboards/${id}/data${query({ to })}`,
    ),

  createWidget: (dashboardId: string, input: WidgetInput) =>
    request<Widget>(`/api/dashboards/${dashboardId}/widgets`, { method: 'POST', ...json(input) }),
  updateWidget: (id: string, input: WidgetInput) =>
    request<Widget>(`/api/widgets/${id}`, { method: 'PUT', ...json(input) }),
  deleteWidget: (id: string) => request<{ ok: true }>(`/api/widgets/${id}`, { method: 'DELETE' }),
  updateLayouts: (dashboardId: string, items: WidgetLayout[]) =>
    request<{ ok: true }>(`/api/dashboards/${dashboardId}/layout`, {
      method: 'PUT',
      ...json({ items }),
    }),
}

export function errorMessage(error: unknown): string {
  if (error instanceof ApiError) return error.message
  if (error instanceof Error) return error.message
  return 'Something went wrong'
}
