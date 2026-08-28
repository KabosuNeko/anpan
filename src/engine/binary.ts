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

/** Async check whether a binary responds to the given args. */
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

/**
 * Resolve a usable yt-dlp binary: system PATH first, then a previously
 * cached copy under ~/.anpan/bin, then download the standalone binary
 * from GitHub releases.
 */
export async function ensureYtDlpBinary(
  onStatus: (message: string) => void,
  signal?: AbortSignal,
): Promise<string> {
  // system install
  if (await binaryResponds('yt-dlp', ['--version'])) return 'yt-dlp'

  // previously cached
  const localBin = path.join(ANPAN_BIN_DIR, process.platform === 'win32' ? 'yt-dlp.exe' : 'yt-dlp')
  if (await binaryResponds(localBin, ['--version'])) return localBin

  // download
  onStatus('preheating oven tools (yt-dlp)…')
  await fs.mkdir(ANPAN_BIN_DIR, {recursive: true})

  const url = `${RELEASE_BASE}/${ytDlpAssetName()}`
  const response = await fetch(url, {signal})
  if (!response.ok || !response.body) {
    throw new Error(`Could not download yt-dlp (${response.status}). Check your connection and try again.`)
  }

  const tmp = `${localBin}.download`
  await pipeline(Readable.fromWeb(response.body as never), createWriteStream(tmp), {signal})
  await fs.chmod(tmp, 0o755)
  await fs.rename(tmp, localBin)
  return localBin
}
