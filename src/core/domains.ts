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
]

/** Identify the video platform from a URL string. */
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

/** Quick check whether a string looks like an http(s) URL. */
export function isLikelyUrl(input: string): boolean {
  try {
    const u = new URL(input.trim())
    return u.protocol === 'http:' || u.protocol === 'https:'
  } catch {
    return false
  }
}
