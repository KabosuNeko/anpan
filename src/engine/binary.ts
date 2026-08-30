import {spawn, type ChildProcess} from 'node:child_process'
import {createWriteStream} from 'node:fs'
import fs from 'node:fs/promises'
import os from 'node:os'
import path from 'node:path'
import {Readable} from 'node:stream'
import {pipeline} from 'node:stream/promises'

const ANPAN_BIN_DIR = path.join(os.homedir(), '.anpan', 'bin')
const RELEASE_BASE = 'https://github.com/yt-dlp/yt-dlp/releases/latest/download'

function ytDlpAssetName(): string {
  if (process.platform === 'win32') return 'yt-dlp.exe'
  if (process.platform === 'darwin') return 'yt-dlp_macos'
  return process.arch === 'arm64' ? 'yt-dlp_linux_aarch64' : 'yt-dlp_linux'
}

// Probing must be asynchronous: a synchronous spawnSync here blocks Node's event loop,
// which freezes Ink mid-render and causes keystroke lag.
export function binaryResponds(cmd: string, args: string[]): Promise<boolean> {
  return new Promise(resolve => {
    let child: ChildProcess
    try {
      child = spawn(cmd, args, {stdio: 'ignore', timeout: 10_000})
    } catch {
      resolve(false)
      return
    }
    child.on('error', () => resolve(false))
    child.on('close', code => resolve(code === 0))
  })
}

let cachedMutagen: boolean | undefined

export async function hasMutagen(ytdlpPath?: string): Promise<boolean> {
  if (ytdlpPath && (ytdlpPath.includes('.anpan') || ytdlpPath.includes('yt-dlp_'))) return true
  const localBin = path.join(ANPAN_BIN_DIR, process.platform === 'win32' ? 'yt-dlp.exe' : 'yt-dlp')
  if (await binaryResponds(localBin, ['--version'])) return true

  if (cachedMutagen !== undefined) return cachedMutagen
  cachedMutagen = await binaryResponds('python3', ['-c', 'import mutagen'])
  return cachedMutagen
}

// Resolves a usable yt-dlp executable:
// Prioritizes ~/.anpan/bin cache (standalone build with mutagen/brotli/crypto bundled)
// -> system yt-dlp if it has mutagen
// -> auto-downloads official standalone binary to ~/.anpan/bin (zero manual dependencies!).
export async function ensureYtDlpBinary(
  onStatus: (message: string) => void,
  signal?: AbortSignal,
): Promise<string> {
  const localBin = path.join(ANPAN_BIN_DIR, process.platform === 'win32' ? 'yt-dlp.exe' : 'yt-dlp')
  if (await binaryResponds(localBin, ['--version'])) return localBin

  const hasSystemYtDlp = await binaryResponds('yt-dlp', ['--version'])
  const sysMutagen = await binaryResponds('python3', ['-c', 'import mutagen'])
  if (hasSystemYtDlp && sysMutagen) return 'yt-dlp'

  onStatus('fetching self-contained yt-dlp (bundled dependencies)…')
  await fs.mkdir(ANPAN_BIN_DIR, {recursive: true})

  const url = `${RELEASE_BASE}/${ytDlpAssetName()}`
  try {
    const response = await fetch(url, {signal})
    if (response.ok && response.body) {
      const tmp = `${localBin}.download`
      await pipeline(Readable.fromWeb(response.body as never), createWriteStream(tmp), {signal})
      await fs.chmod(tmp, 0o755)
      await fs.rename(tmp, localBin)
      return localBin
    }
  } catch {
    // If download fails or offline, fallback to system yt-dlp
  }

  if (hasSystemYtDlp) return 'yt-dlp'
  throw new Error('Could not download standalone yt-dlp. Please check your internet connection.')
}
