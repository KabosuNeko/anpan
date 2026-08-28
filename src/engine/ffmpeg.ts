import {binaryResponds} from './binary.js'

/**
 * Find ffmpeg for stream merging / mp3 extraction.
 * Returns the path to ffmpeg-static if system ffmpeg is missing,
 * or undefined if ffmpeg is already on PATH (yt-dlp finds it itself).
 */
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
