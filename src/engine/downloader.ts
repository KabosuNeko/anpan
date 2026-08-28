import {spawn, type ChildProcess} from 'node:child_process'
import fs from 'node:fs/promises'
import path from 'node:path'
import type {Portion} from './extractor.js'

// ── Types ────────────────────────────────────────────────────────────────────

export type BakeProgress = {
  downloadedBytes: number
  totalBytes?: number
  speed?: number
  eta?: number
  part: number
  totalParts: number
}

export type BakeHandlers = {
  onProgress: (progress: BakeProgress) => void
  onProcessing: () => void
}

// ── Constants ────────────────────────────────────────────────────────────────

const PROGRESS_TAG = 'ANPAN|'
const PROGRESS_TEMPLATE = `${PROGRESS_TAG}%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s`

// Ensure the child process is cleaned up on exit
let activeChild: ChildProcess | undefined
process.on('exit', () => activeChild?.kill('SIGTERM'))

// ── Download ─────────────────────────────────────────────────────────────────

export function bakeVideo(
  opts: {
    ytdlpBin: string
    ffmpegLocation?: string
    aria2cArgs?: string[]
    url: string
    cachedJsonPath?: string
    portion: Portion
    outputDir: string
  },
  handlers: BakeHandlers,
  signal?: AbortSignal,
): Promise<string> {
  const args = [
    ...(opts.cachedJsonPath ? ['--load-info-json', opts.cachedJsonPath] : [opts.url]),
    ...opts.portion.ytdlpArgs,
    '--no-playlist',
    '--no-warnings',
    '--newline',
    '--no-quiet',
    '--progress',
    '--progress-template',
    `download:${PROGRESS_TEMPLATE}`,
    '--print',
    'after_move:filepath',
    '--no-simulate',
    '-o',
    path.join(opts.outputDir, '%(title).60s.%(ext)s'),
  ]
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
    const destinations: string[] = []

    child.stdout.on('data', (chunk: Buffer) => {
      buffer += chunk.toString()
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const rawLine of lines) {
        const line = rawLine.trim()
        if (!line) continue

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
          })
        } else if (line.includes('Downloading 1 format(s):')) {
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
        resolve(filepath)
      } else {
        reject(new Error(cleanErrorOutput(stderr) || `Download failed (yt-dlp exit code ${code}).`))
      }
    })
  })
}

// ── Helpers ──────────────────────────────────────────────────────────────────

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
