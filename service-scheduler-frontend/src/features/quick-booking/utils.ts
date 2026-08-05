export function localDatetimeToOffsetString(dtLocal: string): string {
  // dtLocal like "2026-08-08T08:20" (from <input type="datetime-local">)
  if (!dtLocal) return ''
  const tzMin = -new Date().getTimezoneOffset() // minutes east of UTC
  const sign = tzMin >= 0 ? '+' : '-'
  const absMin = Math.abs(tzMin)
  const tzH = String(Math.floor(absMin / 60)).padStart(2, '0')
  const tzM = String(absMin % 60).padStart(2, '0')
  const offset = `${sign}${tzH}:${tzM}`
  return `${dtLocal}:00${offset}`
}
