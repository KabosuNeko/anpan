import {binaryResponds} from './binary.js'

// Falls back to ffmpeg-static if system ffmpeg is missing.
export async function findFfmpeg(): Promise<string | undefined> {
  if (await binaryResponds('ffmpeg', ['-version'])) return undefined
  try {
    const mod = await import('ffmpeg-static')
    const ffmpegPath = (mod.default ?? mod) as unknown as string | null
    if (ffmpegPath && (await binaryResponds(ffmpegPath, ['-version']))) return ffmpegPath
  } catch {
    // ffmpeg-static not installed or unsupported platform
  }
  return undefined
}
