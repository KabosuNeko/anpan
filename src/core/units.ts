/** Human-readable byte size (e.g. 1024 → "1 KB"). */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return ''
  const tiers = ['B', 'KB', 'MB', 'GB'] as const
  let value = bytes
  let tier = 0
  while (value >= 1024 && tier < tiers.length - 1) {
    value /= 1024
    tier++
  }
  const formatted =
    value >= 10 || tier === 0 || Number.isInteger(value) ? Math.round(value) : value.toFixed(1)
  return `${formatted} ${tiers[tier]}`
}

/** Seconds → "m:ss" or "h:mm:ss". */
export function formatDuration(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) return ''
  const s = Math.round(totalSeconds)
  const hrs = Math.floor(s / 3600)
  const mins = Math.floor((s % 3600) / 60)
  const secs = s % 60
  const mm = hrs > 0 ? String(mins).padStart(2, '0') : String(mins)
  const ss = String(secs).padStart(2, '0')
  return hrs > 0 ? `${hrs}:${mm}:${ss}` : `${mm}:${ss}`
}

/** Bytes/second → human string like "12.4 MB/s". */
export function formatSpeed(bytesPerSecond: number): string {
  if (!Number.isFinite(bytesPerSecond) || bytesPerSecond <= 0) return ''
  return `${formatBytes(bytesPerSecond)}/s`
}

/** Remaining seconds → duration string. */
export function formatEta(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds <= 0) return ''
  return formatDuration(seconds)
}

/** Truncate to `max` columns with trailing "…". */
export function truncate(text: string, max: number): string {
  return text.length > max ? `${text.slice(0, max - 1)}…` : text
}

/** Replace $HOME prefix with ~ and optionally truncate. */
export function shortenPath(filepath: string, homedir: string, max = 60): string {
  const pretty = filepath.startsWith(homedir) ? `~${filepath.slice(homedir.length)}` : filepath
  if (pretty.length <= max) return pretty
  const ext = /\.\w{1,5}$/.exec(pretty)?.[0] ?? ''
  return `${pretty.slice(0, max - ext.length - 1)}…${ext}`
}

/**
 * Word-wrap text into left-flush lines of at most `width` columns.
 * Unlike ink's built-in wrapping, break points don't leave leading spaces
 * on continuation lines.
 */
export function wrapText(text: string, width: number): string[] {
  const lines: string[] = []
  let current = ''
  for (const word of text.split(/\s+/).filter(Boolean)) {
    if (!current) {
      current = word
    } else if (current.length + 1 + word.length <= width) {
      current += ` ${word}`
    } else {
      lines.push(current)
      current = word
    }
  }
  if (current) lines.push(current)
  return lines
}
