import path from 'node:path'
import {identifySite, parseUrlInput} from './domains.js'

export type TargetInspection =
  | {
      type: 'torrent'
      target: string
      name: string
    }
  | {
      type: 'direct'
      url: string
      filename: string
      size?: number
    }
  | {
      type: 'video'
      cleanUrl: string
      timeRange?: string
      timeLabel?: string
    }

const DIRECT_EXTENSIONS = new Set([
  'iso',
  'img',
  'zip',
  'tar',
  'gz',
  'bz2',
  'xz',
  '7z',
  'rar',
  'tgz',
  'bin',
  'pkg',
  'deb',
  'rpm',
  'appimage',
  'exe',
  'dmg',
  'pdf',
  'epub',
  'apk',
  'jar',
])

export function parseMagnetName(magnetUri: string): string {
  try {
    const u = new URL(magnetUri)
    const dn = u.searchParams.get('dn')
    if (dn) return dn.replace(/\+/g, ' ')
    const xt = u.searchParams.get('xt') ?? ''
    const hash = xt.split(':').pop() ?? 'torrent'
    return `Torrent (${hash.slice(0, 10)})`
  } catch {
    return 'BitTorrent Transfer'
  }
}

export function extractFilenameFromUrl(rawUrl: string): string {
  try {
    const u = new URL(rawUrl)
    const base = path.basename(u.pathname)
    return base || 'download'
  } catch {
    return 'download'
  }
}

export async function inspectTarget(
  rawInput: string,
  signal?: AbortSignal,
): Promise<TargetInspection> {
  const trimmed = rawInput.trim()

  // 1. BitTorrent & Magnet
  if (trimmed.startsWith('magnet:?')) {
    return {
      type: 'torrent',
      target: trimmed,
      name: parseMagnetName(trimmed),
    }
  }

  if (trimmed.endsWith('.torrent') || trimmed.includes('.torrent?')) {
    return {
      type: 'torrent',
      target: trimmed,
      name: path.basename(trimmed.split('?')[0]!, '.torrent'),
    }
  }

  // 2. Video / Streaming URL with optional trimming
  const {cleanUrl, timeRange, timeLabel} = parseUrlInput(trimmed)

  // If time range is specified or domain belongs to a known media site, route to video engine
  const site = identifySite(cleanUrl)
  if (timeRange || site.key !== 'generic') {
    return {
      type: 'video',
      cleanUrl,
      timeRange,
      timeLabel,
    }
  }

  // 3. Check for common direct download file extensions
  let pathname = ''
  try {
    pathname = new URL(cleanUrl).pathname.toLowerCase()
  } catch {
    return {type: 'video', cleanUrl}
  }

  const extMatch = /\.([a-z0-9]{2,8})$/i.exec(pathname)
  const ext = extMatch?.[1]?.toLowerCase()
  if (ext && DIRECT_EXTENSIONS.has(ext)) {
    return {
      type: 'direct',
      url: cleanUrl,
      filename: path.basename(pathname),
    }
  }

  // 4. Fast HEAD request for direct attachments / binary streams
  try {
    const headSignal = signal ? AbortSignal.any([signal, AbortSignal.timeout(1500)]) : AbortSignal.timeout(1500)
    const resp = await fetch(cleanUrl, {method: 'HEAD', signal: headSignal})
    if (resp.ok) {
      const disposition = resp.headers.get('content-disposition') ?? ''
      const contentType = (resp.headers.get('content-type') ?? '').toLowerCase()
      const lengthHeader = resp.headers.get('content-length')
      const size = lengthHeader ? Number.parseInt(lengthHeader, 10) : undefined

      const filenameMatch = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(disposition)
      const dispositionFilename = filenameMatch?.[1]

      const isAttachment = disposition.includes('attachment')
      const isBinary =
        contentType.includes('application/octet-stream') ||
        contentType.includes('application/x-') ||
        contentType.includes('application/zip') ||
        (!contentType.includes('text/html') && !contentType.includes('application/xhtml+xml'))

      if ((isAttachment || isBinary) && (dispositionFilename || (size && size > 1_000_000))) {
        return {
          type: 'direct',
          url: cleanUrl,
          filename: dispositionFilename || extractFilenameFromUrl(cleanUrl),
          size: Number.isFinite(size) ? size : undefined,
        }
      }
    }
  } catch {
    // network timeout or HEAD not allowed — fall through to video engine
  }

  return {
    type: 'video',
    cleanUrl,
    timeRange,
    timeLabel,
  }
}
