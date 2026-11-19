/** YYYY-MM-DD helpers for issue due/start dates. */

export function todayISO(d = new Date()): string {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, "0")
  const day = String(d.getDate()).padStart(2, "0")
  return `${y}-${m}-${day}`
}

export function addDaysISO(base: string, days: number): string {
  const d = parseISODate(base) || new Date()
  d.setDate(d.getDate() + days)
  return todayISO(d)
}

export function parseISODate(iso?: string | null): Date | null {
  if (!iso || iso.length < 10) return null
  const d = new Date(`${iso.slice(0, 10)}T00:00:00`)
  return Number.isNaN(d.getTime()) ? null : d
}

export function formatDueLabel(due?: string | null): string {
  if (!due) return ""
  return due.slice(0, 10)
}

/** 未完成且截止日期早于今天 */
export function isOverdue(due?: string | null, status?: string): boolean {
  if (!due || status === "done") return false
  return due.slice(0, 10) < todayISO()
}

export function isDueThisWeek(due?: string | null): boolean {
  if (!due) return false
  const d = due.slice(0, 10)
  const t = todayISO()
  const end = addDaysISO(t, 7)
  return d >= t && d <= end
}

/** Days from rangeStart to date (floor). */
export function dayOffset(rangeStart: string, date: string): number {
  const a = parseISODate(rangeStart)
  const b = parseISODate(date)
  if (!a || !b) return 0
  return Math.round((b.getTime() - a.getTime()) / 86400000)
}

export function enumerateDays(rangeStart: string, dayCount: number): string[] {
  const out: string[] = []
  for (let i = 0; i < dayCount; i++) out.push(addDaysISO(rangeStart, i))
  return out
}
