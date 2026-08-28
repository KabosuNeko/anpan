import {spawn} from 'node:child_process'

/** Detect aria2c on the system PATH. Returns 'aria2c' if found, undefined otherwise. */
export async function findAria2c(): Promise<string | undefined> {
  return new Promise(resolve => {
    let child
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

/** Build yt-dlp flags to use aria2c with configurable connection count. */
export function buildAria2cArgs(aria2cPath: string | undefined, connections = 16): string[] {
  if (!aria2cPath) return []
  const c = Math.max(1, Math.min(32, connections))
  return ['--downloader', 'aria2c', '--downloader-args', `aria2c:-x ${c} -s ${c} -k 1M -j ${c}`]
}
