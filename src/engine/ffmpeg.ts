import fs from 'node:fs/promises'
import path from 'node:path'
import {binaryResponds} from './binary.js'

export async function findFfmpeg(): Promise<string | undefined> {
  const hasSystemFfmpeg = await binaryResponds('ffmpeg', ['-version'])
  const hasSystemFfprobe = await binaryResponds('ffprobe', ['-version'])
  if (hasSystemFfmpeg && hasSystemFfprobe) return undefined

  let ffmpegPath: string | null = null
  let ffprobePath: string | null = null

  try {
    const mod = await import('ffmpeg-static')
    ffmpegPath = (mod.default ?? mod) as unknown as string | null
  } catch {}

  try {
    const mod = await import('ffprobe-static')
    const raw = (mod as unknown as {default?: unknown; path?: string}).default ?? mod
    if (typeof raw === 'string') ffprobePath = raw
    else if (raw && typeof (raw as {path?: unknown}).path === 'string') ffprobePath = (raw as {path: string}).path
    else ffprobePath = raw as unknown as string | null
  } catch {}

  if (ffmpegPath) {
    try {
      await fs.access(ffmpegPath)
    } catch {
      ffmpegPath = null
    }
    if (ffmpegPath && !(await binaryResponds(ffmpegPath, ['-version']))) {
      ffmpegPath = null
    }
  }
  if (ffprobePath) {
    try {
      await fs.access(ffprobePath)
    } catch {
      ffprobePath = null
    }
    if (ffprobePath && !(await binaryResponds(ffprobePath, ['-version']))) {
      ffprobePath = null
    }
  }

  if (!ffmpegPath) return undefined

  const ffmpegDir = path.dirname(ffmpegPath)

  if (ffprobePath) {
    const ffprobeDir = path.dirname(ffprobePath)
    if (ffprobeDir !== ffmpegDir) {
      const dest = path.join(ffmpegDir, path.basename(ffprobePath))
      try {
        await fs.access(dest)
      } catch {
        try {
          await fs.copyFile(ffprobePath, dest)
        } catch {}
      }
      if (process.platform !== 'win32') {
        await fs.chmod(dest, 0o755).catch(() => {})
      }
    }
  }

  return ffmpegDir
}
