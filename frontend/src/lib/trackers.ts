import type { Tracker, TrackerInput } from './types'

export function trackerToInput(tracker: Tracker): TrackerInput {
  return {
    name: tracker.name,
    description: tracker.description,
    icon: tracker.icon,
    color: tracker.color,
    kind: tracker.kind,
    unit: tracker.unit,
    goal: tracker.goal,
    archived: tracker.archived,
  }
}
