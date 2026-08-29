import {spawn, type ChildProcess} from 'node:child_process'
import path from 'node:path'
import type {BakeProgress} from './downloader.js'

export async function findAria2c(): Promise<string | undefined> {
  return new Promise(resolve => {
    let child: ChildProcess
    try {
      child = spawn('aria2c', ['--version'], {stdio: 'ignore', timeout: 5_000})
    } catch {
      resolve(undefined)
      return
    }
    child.on('error', () => resolve(undefined))
    child.on('close', code => resolve(code === 0 ? 'aria2c' : undefined))
  })
}

// Build yt-dlp arguments to delegate socket downloading to aria2c.
// -x: max connections per server
// -s: split file into N segments across connections
// -k 1M: minimum split size to avoid pointless chunk overhead on small transfers
// -j: max concurrent downloads
export function buildAria2cArgs(aria2cPath: string | undefined, connections = 16): string[] {
  if (!aria2cPath) return []
  const c = Math.max(1, Math.min(32, connections))
  return ['--downloader', 'aria2c', '--downloader-args', `aria2c:-x ${c} -s ${c} -k 1M -j ${c}`]
}

export type AriaProgress = {
  downloadedBytes: number
  totalBytes?: number
  speed: number
  eta?: number
  connections: number
}

export function parseUnitBytes(str: string): number {
  const match = /^([0-9.]+)\s*([A-Za-z]+)?$/.exec(str.trim())
  if (!match) return 0
  const val = Number.parseFloat(match[1]!)
  const unit = (match[2] ?? '').toLowerCase()
  if (unit.startsWith('g')) return Math.round(val * 1024 * 1024 * 1024)
  if (unit.startsWith('m')) return Math.round(val * 1024 * 1024)
  if (unit.startsWith('k')) return Math.round(val * 1024)
  return Math.round(val)
}

export function parseAriaEta(etaStr: string): number | undefined {
  let seconds = 0
  const h = /(\d+)h/.exec(etaStr)?.[1]
  const m = /(\d+)m/.exec(etaStr)?.[1]
  const s = /(\d+)s/.exec(etaStr)?.[1]
  if (h) seconds += Number.parseInt(h, 10) * 3600
  if (m) seconds += Number.parseInt(m, 10) * 60
  if (s) seconds += Number.parseInt(s, 10)
  return seconds > 0 ? seconds : undefined
}

export function parseAriaProgressLine(line: string): AriaProgress | null {
  const ariaMatch =
    /^\[#[0-9a-fA-F]+\s+([^\/]+)\/([^(]+)\((\d+)%\)\s+CN:(\d+)\s+DL:([^ \]]+)(?:\s+ETA:([^ \]]+))?\]/.exec(
      line,
    )
  if (!ariaMatch) return null

  const downloadedBytes = parseUnitBytes(ariaMatch[1]!)
  const totalBytes = parseUnitBytes(ariaMatch[2]!)
  const connections = Number.parseInt(ariaMatch[4]!, 10)
  const speed = parseUnitBytes(ariaMatch[5]!)
  let etaSeconds: number | undefined
  if (ariaMatch[6]) {
    etaSeconds = parseAriaEta(ariaMatch[6])
  } else if (speed > 0 && totalBytes > downloadedBytes) {
    etaSeconds = Math.round((totalBytes - downloadedBytes) / speed)
  }

  return {
    downloadedBytes,
    totalBytes: totalBytes > 0 ? totalBytes : undefined,
    connections,
    speed,
    eta: etaSeconds,
  }
}

export function bakeDirectDownload(
  opts: {
    aria2cBin: string
    url: string
    filename?: string
    outputDir: string
    connections?: number
  },
  handlers: {
    onProgress: (progress: BakeProgress) => void
  },
  signal?: AbortSignal,
): Promise<string> {
  const c = Math.max(1, Math.min(32, opts.connections ?? 16))
  const args = [
    '-d',
    opts.outputDir,
    '-x',
    String(c),
    '-s',
    String(c),
    '-k',
    '1M',
    '-j',
    String(c),
    '--summary-interval=1',
    '--auto-file-renaming=false',
    '--allow-overwrite=true',
  ]
  if (opts.filename) {
    args.push('-o', opts.filename)
  }
  args.push(opts.url)

  return runAria2Process(opts.aria2cBin, args, opts.outputDir, opts.filename, handlers, signal)
}

export function bakeTorrentDownload(
  opts: {
    aria2cBin: string
    target: string
    outputDir: string
  },
  handlers: {
    onProgress: (progress: BakeProgress) => void
  },
  signal?: AbortSignal,
): Promise<string> {
  const args = [
    '-d',
    opts.outputDir,
    '--seed-time=0',
    '--summary-interval=1',
    '--bt-stop-timeout=60',
    opts.target,
  ]

  return runAria2Process(opts.aria2cBin, args, opts.outputDir, undefined, handlers, signal)
}

function runAria2Process(
  aria2cBin: string,
  args: string[],
  outputDir: string,
  fallbackFilename: string | undefined,
  handlers: {onProgress: (progress: BakeProgress) => void},
  signal?: AbortSignal,
): Promise<string> {
  return new Promise((resolve, reject) => {
    const child = spawn(aria2cBin, args, {signal})
    let stderr = ''
    let resolvedFile = fallbackFilename ? path.join(outputDir, fallbackFilename) : ''
    let buffer = ''

    child.stdout.on('data', (chunk: Buffer) => {
      buffer += chunk.toString()
      const lines = buffer.split('\n')
      buffer = lines.pop() ?? ''

      for (const rawLine of lines) {
        const line = rawLine.trim()
        if (!line) continue

        const aria = parseAriaProgressLine(line)
        if (aria) {
          handlers.onProgress({
            downloadedBytes: aria.downloadedBytes,
            totalBytes: aria.totalBytes,
            speed: aria.speed,
            eta: aria.eta,
            part: 0,
            totalParts: 1,
          })
        }

        const completeMatch = /\[NOTICE\] Download complete:\s*(.+)$/.exec(line)
        if (completeMatch) {
          resolvedFile = completeMatch[1]!.trim()
        }

        // Parse result table path
        const tableMatch = /^[0-9a-fA-F]+\|OK\s*\|\s*[^|]+\|(.+)$/.exec(line)
        if (tableMatch) {
          resolvedFile = tableMatch[1]!.trim()
        }
      }
    })

    child.stderr.on('data', (chunk: Buffer) => (stderr += chunk))
    child.on('error', reject)
    child.on('close', code => {
      if (signal?.aborted) {
        reject(new Error('Download cancelled.'))
        return
      }
      if (code === 0 && resolvedFile) {
        resolve(resolvedFile)
      } else if (code === 0) {
        resolve(outputDir)
      } else {
        reject(new Error(stderr.trim() || `aria2c exited with code ${code}.`))
      }
    })
  })
}
