import {spawn, type ChildProcess} from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
import {hasMutagen} from './binary.js'
import {parseAriaProgressLine} from './aria2c.js'
import type {Portion} from './extractor.js'

export type BakeProgress = {
  downloadedBytes: number
  totalBytes?: number
  speed?: number
  eta?: number
  part: number
  totalParts: number
  playlistItem?: number
  playlistTotal?: number
  connections?: number
  seeders?: number
}

export type BakeHandlers = {
  onProgress: (progress: BakeProgress) => void
  onProcessing: () => void
}

const PROGRESS_TAG = 'ANPAN|'
const PROGRESS_TEMPLATE = `${PROGRESS_TAG}%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s`

let activeChild: ChildProcess | undefined
process.on('exit', () => activeChild?.kill('SIGTERM'))

export type BakeVideoOptions = {
  ytdlpBin: string
  ffmpegLocation?: string
  aria2cArgs?: string[]
  url: string
  cachedJsonPath?: string
  portion: Portion
  outputDir: string
  timeRange?: string
  isPlaylist?: boolean
  cookiesBrowser?: string
  subtitles?: 'off' | 'embed' | 'write'
  subLangs?: string
  sponsorBlock?: 'off' | 'remove' | 'mark'
  writeThumbnail?: boolean
}

export async function bakeVideo(
  opts: BakeVideoOptions,
  handlers: BakeHandlers,
  signal?: AbortSignal,
): Promise<string> {
  const outputTemplate = opts.isPlaylist
    ? path.join(
        opts.outputDir,
        '%(playlist_title,playlist)s',
        '%(playlist_index)02d - %(title)s.%(ext)s',
      )
    : path.join(
        opts.outputDir,
        `%(title)s${opts.timeRange ? ` [${opts.timeRange.replace(/[:*]/g, '.')}]` : ''}.%(ext)s`,
      )

  // Opus and Ogg require python-mutagen for embedding thumbnails in yt-dlp.
  // If mutagen is not installed, yt-dlp hard-crashes. We gracefully drop --embed-thumbnail
  // so the download succeeds with full audio quality and metadata.
  const isOpusOrOgg =
    opts.portion.label.toLowerCase().includes('opus') ||
    opts.portion.label.toLowerCase().includes('ogg')
  let portionArgs = opts.portion.ytdlpArgs
  if (isOpusOrOgg && !(await hasMutagen(opts.ytdlpBin))) {
    portionArgs = portionArgs.filter(arg => arg !== '--embed-thumbnail')
  }

  const args = [
    ...(opts.cachedJsonPath && !opts.isPlaylist ? ['--load-info-json', opts.cachedJsonPath] : [opts.url]),
    ...portionArgs,
    ...(opts.isPlaylist
      ? [
          '--yes-playlist',
          '--ignore-errors',
          '--replace-in-metadata',
          'playlist_title',
          '(?i)^(?:album|ep|single)\\s*[-:–—]\\s*',
          '',
          '--replace-in-metadata',
          'playlist',
          '(?i)^(?:album|ep|single)\\s*[-:–—]\\s*',
          '',
        ]
      : ['--no-playlist']),
    '--no-warnings',
    '--newline',
    '--no-quiet',
    '--progress',
    '--progress-template',
    `download:${PROGRESS_TEMPLATE}`,
    // --print after_move:filepath writes the resolved final file destination
    // after container merging and postprocessing completes.
    '--print',
    'after_move:filepath',
    '--no-simulate',
    '-o',
    outputTemplate,
  ]
  if (opts.timeRange) {
    args.push('--download-sections', `*${opts.timeRange}`)
  }
  if (opts.ffmpegLocation) args.push('--ffmpeg-location', opts.ffmpegLocation)
  if (opts.aria2cArgs) args.push(...opts.aria2cArgs)

  // Configurable yt-dlp parameters
  if (opts.cookiesBrowser && opts.cookiesBrowser !== 'none') {
    args.push('--cookies-from-browser', opts.cookiesBrowser)
  }
  if (opts.sponsorBlock === 'remove') {
    args.push('--sponsorblock-remove', 'all')
  } else if (opts.sponsorBlock === 'mark') {
    args.push('--sponsorblock-mark', 'all')
  }
  if (opts.subtitles === 'embed') {
    args.push('--embed-subs', '--sub-langs', opts.subLangs || 'vi,en')
  } else if (opts.subtitles === 'write') {
    args.push('--write-subs', '--sub-langs', opts.subLangs || 'vi,en')
  }
  if (opts.writeThumbnail) {
    args.push('--write-thumbnail')
  }

  await fs.mkdir(opts.outputDir, {recursive: true})

  return new Promise((resolve, reject) => {
    const child = spawn(opts.ytdlpBin, args, {signal})
    activeChild = child

    let stderr = ''
    let filepath = ''
    let part = 0
    let totalParts = 1
    let lastDownloaded = 0
    let buffer = ''
    let playlistItem: number | undefined
    let playlistTotal: number | undefined
    const destinations: string[] = []

    child.stdout.on('data', (chunk: Buffer) => {
      buffer += chunk.toString()
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const rawLine of lines) {
        const line = rawLine.trim()
        if (!line) continue

        const itemMatch = /^\[download\] Downloading item (\d+) of (\d+)/.exec(line)
        if (itemMatch) {
          playlistItem = Number.parseInt(itemMatch[1]!, 10)
          playlistTotal = Number.parseInt(itemMatch[2]!, 10)
        }

        // Native yt-dlp progress template
        if (line.startsWith(PROGRESS_TAG)) {
          const [downloaded, total, totalEstimate, speed, eta] = line
            .slice(PROGRESS_TAG.length)
            .split('|')
          const downloadedBytes = parseNumber(downloaded) ?? 0
          if (downloadedBytes < lastDownloaded) part++
          lastDownloaded = downloadedBytes
          handlers.onProgress({
            downloadedBytes,
            totalBytes: parseNumber(total) ?? parseNumber(totalEstimate),
            speed: parseNumber(speed),
            eta: parseNumber(eta),
            part,
            totalParts,
            playlistItem,
            playlistTotal,
          })
        } else {
          const aria = parseAriaProgressLine(line)
          if (aria) {
            if (aria.downloadedBytes < lastDownloaded && aria.downloadedBytes > 0) part++
            lastDownloaded = aria.downloadedBytes
            handlers.onProgress({
              downloadedBytes: aria.downloadedBytes,
              totalBytes: aria.totalBytes,
              speed: aria.speed,
              eta: aria.eta,
              part,
              totalParts,
              playlistItem,
              playlistTotal,
              connections: aria.connections,
              seeders: aria.seeders,
            })
          }
        }

        if (line.includes('Downloading 1 format(s):')) {
          totalParts = (line.split('format(s):')[1] ?? '').trim().split('+').length
        } else if (line.includes('[Merger]') || line.includes('[ExtractAudio]')) {
          const merging = /^\[Merger\] Merging formats into "(.+)"$/.exec(line)?.[1]
          const extracting = /^\[ExtractAudio\] Destination: (.+)$/.exec(line)?.[1]
          const target = merging ?? extracting
          if (target) destinations.push(target)
          handlers.onProcessing()
        } else if (line.startsWith('[download] Destination: ')) {
          destinations.push(line.slice('[download] Destination: '.length))
        } else if (path.isAbsolute(line)) {
          filepath = line
        }
      }
    })

    child.stderr.on('data', (chunk: Buffer) => (stderr += chunk))
    child.on('error', reject)
    child.on('close', code => {
      activeChild = undefined
      if (signal?.aborted) {
        void removePartials(destinations)
        reject(new Error('Download cancelled.'))
        return
      }
      if (code === 0 && filepath) {
        resolve(opts.isPlaylist ? path.dirname(filepath) : filepath)
      } else {
        reject(new Error(cleanErrorOutput(stderr) || `Download failed (yt-dlp exit code ${code}).`))
      }
    })
  })
}

function removePartials(destinations: string[]): Promise<unknown> {
  return Promise.allSettled(
    destinations
      .flatMap(dest => [dest, `${dest}.part`, `${dest}.ytdl`])
      .map(file => fs.rm(file, {force: true})),
  )
}

function parseNumber(value: string | undefined): number | undefined {
  if (!value || value === 'NA' || value === 'None') return undefined
  const n = Number.parseFloat(value)
  return Number.isFinite(n) ? n : undefined
}

function cleanErrorOutput(stderr: string): string {
  const lines = stderr
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.startsWith('ERROR:'))
  const last = lines.at(-1)
  return last ? last.replace(/^ERROR:\s*(\[[^\]]+\]\s*)?/, '') : ''
}
