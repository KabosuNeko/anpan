import {spawn, type ChildProcess} from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
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
}

export type BakeHandlers = {
  onProgress: (progress: BakeProgress) => void
  onProcessing: () => void
}

const PROGRESS_TAG = 'ANPAN|'
const PROGRESS_TEMPLATE = `${PROGRESS_TAG}%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s`

let activeChild: ChildProcess | undefined
process.on('exit', () => activeChild?.kill('SIGTERM'))

export function bakeVideo(
  opts: {
    ytdlpBin: string
    ffmpegLocation?: string
    aria2cArgs?: string[]
    url: string
    cachedJsonPath?: string
    portion: Portion
    outputDir: string
    timeRange?: string
    isPlaylist?: boolean
  },
  handlers: BakeHandlers,
  signal?: AbortSignal,
): Promise<string> {
  const outputTemplate = opts.isPlaylist
    ? path.join(
        opts.outputDir,
        '%(playlist_title|playlist)s',
        '%(playlist_index)02d - %(title).60s.%(ext)s',
      )
    : path.join(
        opts.outputDir,
        `%(title).60s${opts.timeRange ? ` [${opts.timeRange.replace(/[:*]/g, '.')}]` : ''}.%(ext)s`,
      )

  const args = [
    ...(opts.cachedJsonPath && !opts.isPlaylist ? ['--load-info-json', opts.cachedJsonPath] : [opts.url]),
    ...opts.portion.ytdlpArgs,
    ...(opts.isPlaylist ? ['--yes-playlist'] : ['--no-playlist']),
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
          // When delegating to aria2c, yt-dlp yields stdout/stderr to aria2c which logs:
          //   [#7156da 8.6MiB/11MiB(76%) CN:9 DL:9.4MiB ETA:1s]
          // We intercept this so the TUI progress bar and speed indicators remain responsive.
          const ariaMatch =
            /^\[#[0-9a-fA-F]+\s+([^\/]+)\/([^(]+)\((\d+)%\)\s+CN:(\d+)\s+DL:([^ \]]+)(?:\s+ETA:([^ \]]+))?\]/.exec(
              line,
            )
          if (ariaMatch) {
            const downloadedBytes = parseUnitBytes(ariaMatch[1]!)
            const totalBytes = parseUnitBytes(ariaMatch[2]!)
            const speed = parseUnitBytes(ariaMatch[5]!)
            let etaSeconds: number | undefined
            if (ariaMatch[6]) {
              etaSeconds = parseAriaEta(ariaMatch[6])
            } else if (speed > 0 && totalBytes > downloadedBytes) {
              etaSeconds = Math.round((totalBytes - downloadedBytes) / speed)
            }

            if (downloadedBytes < lastDownloaded && downloadedBytes > 0) part++
            lastDownloaded = downloadedBytes
            handlers.onProgress({
              downloadedBytes,
              totalBytes: totalBytes > 0 ? totalBytes : undefined,
              speed,
              eta: etaSeconds,
              part,
              totalParts,
              playlistItem,
              playlistTotal,
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

function parseUnitBytes(str: string): number {
  const match = /^([0-9.]+)\s*([A-Za-z]+)?$/.exec(str.trim())
  if (!match) return 0
  const val = Number.parseFloat(match[1]!)
  const unit = (match[2] ?? '').toLowerCase()
  if (unit.startsWith('g')) return Math.round(val * 1024 * 1024 * 1024)
  if (unit.startsWith('m')) return Math.round(val * 1024 * 1024)
  if (unit.startsWith('k')) return Math.round(val * 1024)
  return Math.round(val)
}

function parseAriaEta(etaStr: string): number | undefined {
  let seconds = 0
  const h = /(\d+)h/.exec(etaStr)?.[1]
  const m = /(\d+)m/.exec(etaStr)?.[1]
  const s = /(\d+)s/.exec(etaStr)?.[1]
  if (h) seconds += Number.parseInt(h, 10) * 3600
  if (m) seconds += Number.parseInt(m, 10) * 60
  if (s) seconds += Number.parseInt(s, 10)
  return seconds > 0 ? seconds : undefined
}

function cleanErrorOutput(stderr: string): string {
  const lines = stderr
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.startsWith('ERROR:'))
  const last = lines.at(-1)
  return last ? last.replace(/^ERROR:\s*(\[[^\]]+\]\s*)?/, '') : ''
}
