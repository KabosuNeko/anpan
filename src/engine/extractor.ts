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
  fps?: number
  dynamic_range?: string
  vbr?: number
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

export type ExtractPortionsOptions = {
  embedMetadata?: boolean
  videoContainer?: 'mp4' | 'mkv' | 'webm'
  audioFormat?: 'mp3' | 'm4a' | 'opus' | 'flac' | 'wav'
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

export function extractPortions(meta: VideoMeta, opts?: ExtractPortionsOptions): Portion[] {
  const streams = meta.formats ?? []
  const portions: Portion[] = []
  const container = opts?.videoContainer ?? 'mp4'
  const audioFmt = opts?.audioFormat ?? 'mp3'

  const audioStreams = streams.filter(
    s => s.acodec && s.acodec !== 'none' && (!s.vcodec || s.vcodec === 'none'),
  )
  const bestAudio = [...audioStreams].sort((a, b) => (b.abr ?? b.tbr ?? 0) - (a.abr ?? a.tbr ?? 0))[0]
  let audioSize = bestAudio?.filesize ?? bestAudio?.filesize_approx
  if (!audioSize && (bestAudio?.abr || bestAudio?.tbr) && meta.duration) {
    audioSize = Math.round((((bestAudio.abr ?? bestAudio.tbr ?? 128) * 1000) / 8) * meta.duration)
  }

  const videoStreams = streams.filter(s => s.vcodec && s.vcodec !== 'none' && s.height)
  // Retrieve ALL available heights in descending order (no truncation to 8 tiers)
  const heights = [...new Set(videoStreams.map(s => s.height as number))].sort((a, b) => b - a)

  for (const height of heights) {
    const candidates = videoStreams.filter(s => s.height === height)
    const best = [...candidates].sort((a, b) => scoreStream(b, container) - scoreStream(a, container))[0]!
    const isMuxed = best.acodec !== undefined && best.acodec !== 'none'

    // Calculate video bytes from filesize, filesize_approx, or tbr/vbr
    let videoBytes = best.filesize ?? best.filesize_approx
    if (!videoBytes && (best.tbr || best.vbr) && meta.duration) {
      const rate = best.tbr ?? (best.vbr ?? 0) + (best.abr ?? 0)
      videoBytes = Math.round(((rate * 1000) / 8) * meta.duration)
    }

    const estimatedSize = videoBytes ? videoBytes + (isMuxed ? 0 : audioSize ?? 0) : 0
    const sizeTag = estimatedSize > 0 ? ` · ~${formatBytes(estimatedSize)}` : ''

    // Framerate & HDR tags
    const fpsTag = best.fps && best.fps >= 50 ? `${Math.round(best.fps)}` : ''
    const hdrTag = best.dynamic_range && best.dynamic_range.toLowerCase().includes('hdr') ? ' HDR' : ''

    portions.push({
      kind: 'video',
      label: `${height}p${fpsTag}${hdrTag} · ${container}${sizeTag}`,
      ytdlpArgs: [
        '-f',
        `bv*[height=${height}]+ba/b[height=${height}]/bv*[height<=${height}]+ba/b`,
        '--merge-output-format',
        container,
      ],
    })
  }

  if (portions.length === 0) {
    portions.push({
      kind: 'video',
      label: `best available · ${container}`,
      ytdlpArgs: ['-f', 'bv*+ba/b', '--merge-output-format', container],
    })
  }

  // Audio streams
  const addedAudioFormats = new Set<string>()

  // 1. Check for native Opus stream (YouTube / YTM native audio)
  const nativeOpus = audioStreams.find(
    s => s.acodec?.toLowerCase().includes('opus') || s.ext === 'webm',
  )
  if (nativeOpus) {
    let opusBytes = nativeOpus.filesize ?? nativeOpus.filesize_approx
    if (!opusBytes && (nativeOpus.abr || nativeOpus.tbr) && meta.duration) {
      opusBytes = Math.round((((nativeOpus.abr ?? nativeOpus.tbr ?? 128) * 1000) / 8) * meta.duration)
    }
    const opusSizeTag = opusBytes ? ` · ~${formatBytes(opusBytes)}` : ''
    const opusAbrTag = nativeOpus.abr ? ` · ~${Math.round(nativeOpus.abr)}kbps` : ''
    const opusArgs = ['-f', 'ba[acodec^=opus]/ba', '-x', '--audio-format', 'opus']
    if (opts?.embedMetadata !== false) opusArgs.push('--embed-thumbnail', '--add-metadata')
    portions.push({
      kind: 'audio',
      label: `audio only · opus (original)${opusAbrTag}${opusSizeTag}`,
      ytdlpArgs: opusArgs,
    })
    addedAudioFormats.add('opus')
  }

  // 2. Check for native M4A / AAC stream
  const nativeM4a = audioStreams.find(
    s => s.ext === 'm4a' || s.acodec?.toLowerCase().includes('mp4a'),
  )
  if (nativeM4a) {
    let m4aBytes = nativeM4a.filesize ?? nativeM4a.filesize_approx
    if (!m4aBytes && (nativeM4a.abr || nativeM4a.tbr) && meta.duration) {
      m4aBytes = Math.round((((nativeM4a.abr ?? nativeM4a.tbr ?? 128) * 1000) / 8) * meta.duration)
    }
    const m4aSizeTag = m4aBytes ? ` · ~${formatBytes(m4aBytes)}` : ''
    const m4aAbrTag = nativeM4a.abr ? ` · ~${Math.round(nativeM4a.abr)}kbps` : ''
    const m4aArgs = ['-f', 'ba[ext=m4a]/ba', '-x', '--audio-format', 'm4a']
    if (opts?.embedMetadata !== false) m4aArgs.push('--embed-thumbnail', '--add-metadata')
    portions.push({
      kind: 'audio',
      label: `audio only · m4a (aac)${m4aAbrTag}${m4aSizeTag}`,
      ytdlpArgs: m4aArgs,
    })
    addedAudioFormats.add('m4a')
  }

  // 3. User configured audio format (if not already added as opus/m4a)
  if (!addedAudioFormats.has(audioFmt)) {
    const audioSizeTag = audioSize ? ` · ~${formatBytes(audioSize)}` : ''
    const audioBitrateTag = bestAudio?.abr ? ` · ${Math.round(bestAudio.abr)}kbps` : ''
    const audioQuality = audioFmt === 'flac' || audioFmt === 'wav' ? '0' : '0'
    const audioArgs = ['-f', 'ba/b', '-x', '--audio-format', audioFmt, '--audio-quality', audioQuality]
    if (opts?.embedMetadata !== false) audioArgs.push('--embed-thumbnail', '--add-metadata')
    portions.push({
      kind: 'audio',
      label: `audio only · ${audioFmt}${audioBitrateTag}${audioSizeTag}`,
      ytdlpArgs: audioArgs,
    })
    addedAudioFormats.add(audioFmt)
  }

  // 4. Universal MP3 fallback if not yet added
  if (!addedAudioFormats.has('mp3')) {
    const mp3SizeTag = audioSize ? ` · ~${formatBytes(audioSize)}` : ''
    const mp3Args = ['-f', 'ba/b', '-x', '--audio-format', 'mp3', '--audio-quality', '0']
    if (opts?.embedMetadata !== false) mp3Args.push('--embed-thumbnail', '--add-metadata')
    portions.push({
      kind: 'audio',
      label: `audio only · mp3 · 320kbps${mp3SizeTag}`,
      ytdlpArgs: mp3Args,
    })
    addedAudioFormats.add('mp3')
  }

  return portions
}

export type PlaylistMeta = {
  title: string
  uploader?: string
  trackCount: number
  webpage_url: string
}

// Light probe for playlist info using --flat-playlist: returns in ~1s without pulling full stream lists
export async function probePlaylist(
  ytdlpBin: string,
  url: string,
  signal?: AbortSignal,
): Promise<PlaylistMeta | null> {
  try {
    const stdout = await new Promise<string>((resolve, reject) => {
      const child = spawn(
        ytdlpBin,
        ['--flat-playlist', '-J', '--no-warnings', url],
        {signal},
      )
      let out = ''
      let errOut = ''
      child.stdout.on('data', chunk => (out += chunk))
      child.stderr.on('data', chunk => (errOut += chunk))
      child.on('error', reject)
      child.on('close', code => {
        if (code !== 0) reject(new Error(cleanErrorOutput(errOut) || `yt-dlp exited with code ${code}`))
        else resolve(out)
      })
    })

    const data = JSON.parse(stdout) as {
      _type?: string
      title?: string
      uploader?: string
      playlist_count?: number
      entries?: unknown[]
      webpage_url?: string
    }

    if (data._type === 'playlist' || Array.isArray(data.entries)) {
      const trackCount = data.playlist_count ?? data.entries?.length ?? 0
      return {
        title: data.title || 'Playlist',
        uploader: data.uploader,
        trackCount,
        webpage_url: data.webpage_url || url,
      }
    }
    return null
  } catch {
    return null
  }
}

export function extractPlaylistPortions(opts?: ExtractPortionsOptions): Portion[] {
  const container = opts?.videoContainer ?? 'mp4'
  const audioFmt = opts?.audioFormat ?? 'mp3'

  const audioQuality = audioFmt === 'flac' || audioFmt === 'wav' ? '0' : '0'
  const audioArgs = ['-f', 'ba/b', '-x', '--audio-format', audioFmt, '--audio-quality', audioQuality]
  if (opts?.embedMetadata !== false) {
    audioArgs.push('--embed-thumbnail', '--add-metadata')
  }
  const portions: Portion[] = [
    {
      kind: 'audio',
      label: `all tracks · ${audioFmt} (audio only)`,
      ytdlpArgs: audioArgs,
    },
  ]

  if (audioFmt !== 'opus') {
    const opusArgs = ['-f', 'ba[acodec^=opus]/ba', '-x', '--audio-format', 'opus']
    if (opts?.embedMetadata !== false) opusArgs.push('--embed-thumbnail', '--add-metadata')
    portions.push({
      kind: 'audio',
      label: 'all tracks · opus (original audio)',
      ytdlpArgs: opusArgs,
    })
  }

  if (audioFmt !== 'mp3') {
    const mp3Args = ['-f', 'ba/b', '-x', '--audio-format', 'mp3', '--audio-quality', '0']
    if (opts?.embedMetadata !== false) mp3Args.push('--embed-thumbnail', '--add-metadata')
    portions.push({
      kind: 'audio',
      label: 'all tracks · mp3 (audio only)',
      ytdlpArgs: mp3Args,
    })
  }

  portions.push({
    kind: 'video',
    label: `all videos · ${container} (best quality)`,
    ytdlpArgs: ['-f', 'bv*+ba/b', '--merge-output-format', container],
  })

  return portions
}

function scoreStream(s: RawStream, preferredContainer = 'mp4'): number {
  let score = s.tbr ?? 0
  if (s.ext === preferredContainer) score += 10_000
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
