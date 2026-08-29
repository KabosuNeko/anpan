import {spawn} from 'node:child_process'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import {formatBytes} from '../core/units.js'

export type VideoMeta = {
  title: string
  uploader?: string
  duration?: number
  webpage_url?: string
  extractor_key?: string
  formats?: RawStream[]
}

type RawStream = {
  format_id: string
  ext?: string
  vcodec?: string
  acodec?: string
  height?: number
  width?: number
  abr?: number
  tbr?: number
  filesize?: number
  filesize_approx?: number
}

export type ProbeResult = {
  meta: VideoMeta
  cachedJsonPath: string
}

export type Portion = {
  label: string
  kind: 'video' | 'audio'
  ytdlpArgs: string[]
}

export async function probeVideo(
  ytdlpBin: string,
  url: string,
  signal?: AbortSignal,
): Promise<ProbeResult> {
  const stdout = await new Promise<string>((resolve, reject) => {
    const child = spawn(ytdlpBin, ['-J', '--no-playlist', '--no-warnings', url], {signal})
    let out = ''
    let errOut = ''
    child.stdout.on('data', chunk => (out += chunk))
    child.stderr.on('data', chunk => (errOut += chunk))
    child.on('error', reject)
    child.on('close', code => {
      if (code !== 0) {
        reject(new Error(cleanErrorOutput(errOut) || `yt-dlp exited with code ${code}`))
      } else {
        resolve(out)
      }
    })
  })

  let meta: VideoMeta
  try {
    meta = JSON.parse(stdout) as VideoMeta
  } catch {
    throw new Error('Could not parse video info from yt-dlp.')
  }

  // Dump full -J JSON to a temp file so bakeVideo can invoke --load-info-json.
  // Re-extracting from YouTube/X on download start costs 2-4 seconds of redundant network latency.
  const cachedJsonPath = path.join(os.tmpdir(), `anpan-meta-${process.pid}-${Date.now()}.json`)
  await fs.writeFile(cachedJsonPath, stdout)
  return {meta, cachedJsonPath}
}

const MAX_VIDEO_TIERS = 8

export function extractPortions(meta: VideoMeta): Portion[] {
  const streams = meta.formats ?? []
  const portions: Portion[] = []

  const audioStreams = streams.filter(
    s => s.acodec && s.acodec !== 'none' && (!s.vcodec || s.vcodec === 'none'),
  )
  const bestAudio = [...audioStreams].sort((a, b) => (b.abr ?? b.tbr ?? 0) - (a.abr ?? a.tbr ?? 0))[0]
  const audioSize = bestAudio?.filesize ?? bestAudio?.filesize_approx

  const videoStreams = streams.filter(s => s.vcodec && s.vcodec !== 'none' && s.height)
  const heights = [...new Set(videoStreams.map(s => s.height as number))].sort((a, b) => b - a)

  for (const height of heights.slice(0, MAX_VIDEO_TIERS)) {
    const candidates = videoStreams.filter(s => s.height === height)
    const best = [...candidates].sort((a, b) => scoreStream(b) - scoreStream(a))[0]!
    const isMuxed = best.acodec !== undefined && best.acodec !== 'none'

    // Adaptive DASH/HLS streams rarely provide 'filesize' in metadata.
    // Fall back to estimating bytes from total bitrate (tbr in kbps) and duration.
    let videoBytes = best.filesize ?? best.filesize_approx
    if (!videoBytes && best.tbr && meta.duration) {
      videoBytes = Math.round(((best.tbr * 1000) / 8) * meta.duration)
    }

    const estimatedSize = videoBytes ? videoBytes + (isMuxed ? 0 : audioSize ?? 0) : 0
    const sizeTag = estimatedSize > 0 ? ` · ~${formatBytes(estimatedSize)}` : ''
    portions.push({
      kind: 'video',
      label: `${height}p · mp4${sizeTag}`,
      ytdlpArgs: [
        '-f',
        `bv*[height=${height}]+ba/b[height=${height}]/bv*[height<=${height}]+ba/b`,
        '--merge-output-format',
        'mp4',
      ],
    })
  }

  if (portions.length === 0) {
    portions.push({
      kind: 'video',
      label: 'best available · mp4',
      ytdlpArgs: ['-f', 'bv*+ba/b', '--merge-output-format', 'mp4'],
    })
  }

  const audioSizeTag = audioSize ? ` · ~${formatBytes(audioSize)}` : ''
  portions.push({
    kind: 'audio',
    label: `audio only · mp3${audioSizeTag}`,
    ytdlpArgs: ['-f', 'ba/b', '-x', '--audio-format', 'mp3', '--audio-quality', '0'],
  })

  return portions
}

function scoreStream(s: RawStream): number {
  let score = s.tbr ?? 0
  if (s.ext === 'mp4') score += 10_000
  if (s.vcodec?.startsWith('avc')) score += 5_000
  return score
}

function cleanErrorOutput(stderr: string): string {
  const lines = stderr
    .split('\n')
    .map(l => l.trim())
    .filter(l => l.startsWith('ERROR:'))
  const last = lines.at(-1)
  return last ? last.replace(/^ERROR:\s*(\[[^\]]+\]\s*)?/, '') : ''
}
