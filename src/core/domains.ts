export type SiteInfo = {
  key: string
  label: string
}

const KNOWN_SITES: ReadonlyArray<{hosts: string[]; site: SiteInfo}> = [
  {hosts: ['youtube.com', 'youtu.be', 'music.youtube.com'], site: {key: 'youtube', label: 'YouTube'}},
  {hosts: ['x.com', 'twitter.com'], site: {key: 'x', label: 'X / Twitter'}},
  {hosts: ['instagram.com'], site: {key: 'instagram', label: 'Instagram'}},
  {hosts: ['threads.net', 'threads.com'], site: {key: 'threads', label: 'Threads'}},
  {hosts: ['tiktok.com'], site: {key: 'tiktok', label: 'TikTok'}},
  {hosts: ['vimeo.com'], site: {key: 'vimeo', label: 'Vimeo'}},
  {hosts: ['twitch.tv'], site: {key: 'twitch', label: 'Twitch'}},
  {hosts: ['reddit.com'], site: {key: 'reddit', label: 'Reddit'}},
  {hosts: ['facebook.com', 'fb.watch'], site: {key: 'facebook', label: 'Facebook'}},
  {hosts: ['soundcloud.com'], site: {key: 'soundcloud', label: 'SoundCloud'}},
  {hosts: ['bandcamp.com'], site: {key: 'bandcamp', label: 'Bandcamp'}},
  {hosts: ['kemono.cr', 'kemono.su', 'kemono.party'], site: {key: 'kemono', label: 'Kemono'}},
  {hosts: ['coomer.su', 'coomer.party', 'coomer.st'], site: {key: 'coomer', label: 'Coomer'}},
  {hosts: ['pawchive.st', 'pawchive.pw'], site: {key: 'pawchive', label: 'Pawchive'}},
]

export function identifySite(url: string): SiteInfo {
  let hostname: string
  try {
    hostname = new URL(url).hostname.toLowerCase()
  } catch {
    return {key: 'unknown', label: 'Unknown site'}
  }

  for (const {hosts, site} of KNOWN_SITES) {
    if (hosts.some(h => hostname === h || hostname.endsWith(`.${h}`))) {
      return site
    }
  }

  return {key: 'generic', label: hostname}
}

const TIME_RANGE_RE = /(?:^|\s+)((?:\d{1,2}:)?\d{1,2}:\d{2}|\d+)\s*-\s*((?:\d{1,2}:)?\d{1,2}:\d{2}|\d+)\s*$/

export type ParsedInput = {
  cleanUrl: string
  timeRange?: string
  timeLabel?: string
}

export function parseUrlInput(input: string): ParsedInput {
  const trimmed = input.trim()
  const match = TIME_RANGE_RE.exec(trimmed)
  if (!match) {
    return {cleanUrl: trimmed}
  }

  const start = match[1]!
  const end = match[2]!
  const cleanUrl = trimmed.slice(0, match.index).trim()
  return {
    cleanUrl,
    timeRange: `${start}-${end}`,
    timeLabel: `${start} → ${end}`,
  }
}

export function isLikelyUrl(input: string): boolean {
  try {
    const {cleanUrl} = parseUrlInput(input)
    const u = new URL(cleanUrl)
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}

export function isLikelyTarget(input: string): boolean {
  const trimmed = input.trim()
  if (trimmed.startsWith('magnet:?')) return true
  if (trimmed.endsWith('.torrent') || trimmed.includes('.torrent?')) return true
  return isLikelyUrl(trimmed)
}

export function isPlaylistUrl(url: string): boolean {
  try {
    const u = new URL(url)
    if (u.searchParams.has('list')) return true
    if (u.pathname.includes('/playlist') || u.pathname.includes('/sets/')) return true
    return false
  } catch {
    return false
  }
}
