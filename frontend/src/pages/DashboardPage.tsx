import { useEffect, useState, type FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowRight,
  Check,
  EllipsisVertical,
  LayoutDashboard,
  Pencil,
  Plus,
  Trash,
} from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { DashboardGrid } from '../components/dashboard/DashboardGrid'
import { WidgetFormModal } from '../components/dashboard/WidgetFormModal'
import { ConfirmDialog } from '../components/ui/ConfirmDialog'
import { EmptyState } from '../components/ui/EmptyState'
import { ErrorState } from '../components/ui/ErrorState'
import { Modal } from '../components/ui/Modal'
import { Spinner } from '../components/ui/Spinner'
import { useToast } from '../components/ui/Toast'
import { api, errorMessage } from '../lib/api'
import { todayKey } from '../lib/date'
import { useDashboardData, useDashboards, useTrackers } from '../lib/queries'
import { useMediaQuery } from '../lib/useMediaQuery'
import type { Widget } from '../lib/types'

type NameModal = { mode: 'create' | 'rename'; value: string }

export function DashboardPage() {
  const { dashboardId } = useParams()
  const navigate = useNavigate()
  const toast = useToast()
  const queryClient = useQueryClient()
  const mobile = useMediaQuery('(max-width: 1023px)')
  const today = todayKey()

  const { data: dashboards = [], isLoading, isError, error, refetch, isFetching } = useDashboards()
  const { data: trackers = [] } = useTrackers(false)

  const [editing, setEditing] = useState(false)
  const [widgetModal, setWidgetModal] = useState<{ open: boolean; widget: Widget | null }>({
    open: false,
    widget: null,
  })
  const [deletingWidget, setDeletingWidget] = useState<Widget | null>(null)
  const [nameModal, setNameModal] = useState<NameModal | null>(null)
  const [deleteDashboardOpen, setDeleteDashboardOpen] = useState(false)

  const active =
    dashboards.find((dashboard) => dashboard.id === dashboardId) ??
    dashboards.find((dashboard) => dashboard.isDefault) ??
    dashboards[0]

  const {
    data: dashboardData,
    isLoading: dataLoading,
    isError: dataError,
    error: dataErrorValue,
    refetch: refetchData,
    isFetching: dataFetching,
  } = useDashboardData(active?.id, today)

  useEffect(() => {
    if (!dashboardId && active) navigate(`/dashboard/${active.id}`, { replace: true })
  }, [dashboardId, active, navigate])

  useEffect(() => {
    document.title = active ? `${active.name} · Trackete` : 'Dashboard · Trackete'
  }, [active])

  const createDashboard = useMutation({
    mutationFn: (name: string) => api.createDashboard(name),
    onSuccess: (dashboard) => {
      void queryClient.invalidateQueries({ queryKey: ['dashboards'] })
      navigate(`/dashboard/${dashboard.id}`)
      toast.push('success', 'Dashboard created.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const renameDashboard = useMutation({
    mutationFn: (args: { id: string; name: string }) => api.renameDashboard(args.id, args.name),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dashboards'] })
      toast.push('success', 'Dashboard renamed.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const removeDashboard = useMutation({
    mutationFn: (id: string) => api.deleteDashboard(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dashboards'] })
      setDeleteDashboardOpen(false)
      navigate('/dashboard', { replace: true })
      toast.push('success', 'Dashboard deleted.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const removeWidget = useMutation({
    mutationFn: (id: string) => api.deleteWidget(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['dashboards'] })
      void queryClient.invalidateQueries({ queryKey: ['dashboard-data'] })
      setDeletingWidget(null)
      toast.push('success', 'Chart removed.')
    },
    onError: (error) => toast.push('error', errorMessage(error)),
  })

  const onSubmitName = (event: FormEvent) => {
    event.preventDefault()
    if (!nameModal) return
    const name = nameModal.value.trim()
    if (!name) return
    if (nameModal.mode === 'create') {
      createDashboard.mutate(name)
    } else if (active) {
      renameDashboard.mutate({ id: active.id, name })
    }
    setNameModal(null)
  }

  if (isLoading) {
    return <Spinner label="Loading dashboards…" />
  }

  if (isError) {
    return (
      <ErrorState
        title="Couldn't load your dashboards"
        description={errorMessage(error)}
        onRetry={() => void refetch()}
        retrying={isFetching}
      />
    )
  }

  if (!active) {
    return (
      <ErrorState
        title="No dashboards yet"
        description="Your account has no dashboards. Try again, or create one from the menu on any dashboard."
        onRetry={() => void refetch()}
        retrying={isFetching}
      />
    )
  }

  const widgets = active.widgets

  return (
    <div className="space-y-6">
      <header className="flex flex-col gap-4 xl:flex-row xl:items-end xl:justify-between">
        <div className="min-w-0">
          <p className="text-xs font-medium uppercase tracking-wider text-base-content/50">Dashboard</p>
          <div className="mt-1 flex items-center gap-2">
            <h1 className="truncate text-2xl font-semibold tracking-tight sm:text-3xl">{active.name}</h1>
            {active.isDefault && <span className="badge badge-ghost badge-sm">default</span>}
          </div>
          <p className="mt-1 text-sm text-base-content/60">
            {widgets.length === 0
              ? 'Add charts to turn your logs into a story.'
              : editing
                ? 'Drag cards by their header, resize from the bottom-right corner.'
                : `${widgets.length} chart${widgets.length === 1 ? '' : 's'} · switch on edit mode to rearrange.`}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          {dashboards.length > 1 && (
            <select
              className="select select-bordered select-sm"
              value={active.id}
              onChange={(event) => navigate(`/dashboard/${event.target.value}`)}
              aria-label="Switch dashboard"
            >
              {dashboards.map((dashboard) => (
                <option key={dashboard.id} value={dashboard.id}>
                  {dashboard.name}
                </option>
              ))}
            </select>
          )}

          <button
            type="button"
            className={`btn btn-sm ${editing ? 'btn-primary' : 'btn-outline'}`}
            onClick={() => setEditing((value) => !value)}
          >
            {editing ? <Check className="size-4" /> : <Pencil className="size-3.5" />}
            {editing ? 'Done editing' : 'Edit layout'}
          </button>

          <button
            type="button"
            className="btn btn-sm btn-primary"
            onClick={() => setWidgetModal({ open: true, widget: null })}
            disabled={trackers.length === 0}
            title={trackers.length === 0 ? 'Create a tracker first' : 'Add a chart'}
          >
            <Plus className="size-4" />
            Add chart
          </button>

          <div className="dropdown dropdown-end">
            <button
              type="button"
              tabIndex={0}
              className="btn btn-sm btn-ghost btn-square"
              aria-label="Dashboard actions"
            >
              <EllipsisVertical className="size-4" />
            </button>
            <ul
              tabIndex={0}
              className="menu dropdown-content z-40 mt-2 w-52 rounded-box border border-base-300 bg-base-100 p-2 shadow-lg"
            >
              <li>
                <button
                  type="button"
                  onClick={() => {
                    setNameModal({ mode: 'create', value: '' })
                    ;(document.activeElement as HTMLElement | null)?.blur()
                  }}
                >
                  <Plus className="size-4" />
                  New dashboard
                </button>
              </li>
              <li>
                <button
                  type="button"
                  onClick={() => {
                    setNameModal({ mode: 'rename', value: active.name })
                    ;(document.activeElement as HTMLElement | null)?.blur()
                  }}
                >
                  <Pencil className="size-4" />
                  Rename dashboard
                </button>
              </li>
              <li>
                <button
                  type="button"
                  className="text-error"
                  onClick={() => {
                    setDeleteDashboardOpen(true)
                    ;(document.activeElement as HTMLElement | null)?.blur()
                  }}
                >
                  <Trash className="size-4" />
                  Delete dashboard
                </button>
              </li>
            </ul>
          </div>
        </div>
      </header>

      {trackers.length === 0 ? (
        <EmptyState
          icon={LayoutDashboard}
          title="Create a tracker to chart it"
          description="Dashboards are made of charts, and every chart is built on one tracker. Start by tracking something — a yes/no habit or a number."
          action={
            <Link to="/trackers" className="btn btn-primary">
              <Plus className="size-4" />
              Create a tracker
              <ArrowRight className="size-4" />
            </Link>
          }
        />
      ) : widgets.length === 0 ? (
        <EmptyState
          icon={LayoutDashboard}
          title="This dashboard is empty"
          description="Add your first chart: pick a tracker, pick a chart type, choose how much history to show."
          action={
            <button
              type="button"
              className="btn btn-primary"
              onClick={() => setWidgetModal({ open: true, widget: null })}
            >
              <Plus className="size-4" />
              Add your first chart
            </button>
          }
        />
      ) : (
        <div className="space-y-4">
          {dataError && (
            <div role="alert" className="alert alert-error alert-soft flex flex-wrap items-center gap-3 py-2 text-sm">
              <span className="flex-1">Couldn't load chart data. {errorMessage(dataErrorValue)}</span>
              <button
                type="button"
                className="btn btn-xs btn-outline"
                onClick={() => void refetchData()}
                disabled={dataFetching}
              >
                {dataFetching && <span className="loading loading-spinner loading-xs" />}
                Retry
              </button>
            </div>
          )}
          <DashboardGrid
            dashboard={active}
            data={dashboardData?.widgets}
            loading={dataLoading}
            editing={editing}
            mobile={mobile}
            trackers={trackers}
            onEditWidget={(widget) => setWidgetModal({ open: true, widget })}
            onDeleteWidget={setDeletingWidget}
          />
        </div>
      )}

      {widgetModal.open && (
        <WidgetFormModal
          dashboardId={active.id}
          widget={widgetModal.widget}
          trackers={trackers}
          onClose={() => setWidgetModal({ open: false, widget: null })}
        />
      )}

      <ConfirmDialog
        open={Boolean(deletingWidget)}
        title="Remove this chart?"
        message="The tracker and its entries stay untouched — only the chart is removed from this dashboard."
        confirmLabel="Remove chart"
        busy={removeWidget.isPending}
        onConfirm={() => deletingWidget && removeWidget.mutate(deletingWidget.id)}
        onClose={() => setDeletingWidget(null)}
      />

      <ConfirmDialog
        open={deleteDashboardOpen}
        title={`Delete “${active.name}”?`}
        message="This removes the dashboard and all of its charts. Your trackers and entries are kept."
        busy={removeDashboard.isPending}
        onConfirm={() => removeDashboard.mutate(active.id)}
        onClose={() => setDeleteDashboardOpen(false)}
      />

      <Modal
        open={Boolean(nameModal)}
        title={nameModal?.mode === 'create' ? 'New dashboard' : 'Rename dashboard'}
        onClose={() => setNameModal(null)}
        footer={
          <>
            <button type="button" className="btn btn-ghost" onClick={() => setNameModal(null)}>
              Cancel
            </button>
            <button
              type="submit"
              form="dashboard-name-form"
              className="btn btn-primary"
              disabled={createDashboard.isPending || renameDashboard.isPending}
            >
              {nameModal?.mode === 'create' ? 'Create' : 'Save'}
            </button>
          </>
        }
      >
        <form id="dashboard-name-form" onSubmit={onSubmitName}>
          <label className="form-control">
            <span className="mb-1.5 block text-sm font-medium">Dashboard name</span>
            <input
              className="input input-bordered w-full"
              value={nameModal?.value ?? ''}
              onChange={(event) =>
                setNameModal((current) => (current ? { ...current, value: event.target.value } : current))
              }
              maxLength={60}
              required
              autoFocus
            />
          </label>
        </form>
      </Modal>
    </div>
  )
}
