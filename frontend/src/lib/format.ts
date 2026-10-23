export function formatNumber(value: number): string {
  if (!Number.isFinite(value)) return '0'
  const rounded = Math.round(value * 100) / 100
  if (Number.isInteger(rounded)) return rounded.toLocaleString()
  return rounded.toLocaleString(undefined, { maximumFractionDigits: 2 })
}

function unitFor(value: number, unit: string): string {
  // "1 beers" reads badly; pluralise the common case without a dictionary.
  if (Math.abs(value) === 1 && unit.endsWith('s') && !unit.endsWith('ss')) {
    return unit.slice(0, -1)
  }
  return unit
}

export function formatValue(value: number, unit?: string, kind?: string): string {
  if (kind === 'boolean') return value >= 0.5 ? 'Yes' : 'No'
  const text = formatNumber(value)
  return unit ? `${text} ${unitFor(value, unit)}` : text
}

export function formatPercent(ratio: number): string {
  return `${Math.round(ratio * 100)}%`
}

export function goalLabel(
  goal: { type: string; value: number; period: string } | null,
  kind: string,
  unit: string,
): string | null {
  if (!goal) return null
  const comparator = goal.type === 'at_most' ? '≤' : '≥'
  const amount = kind === 'boolean' ? `${formatNumber(goal.value)}×` : formatValue(goal.value, unit, kind)
  return `${comparator} ${amount} / ${goal.period}`
}
