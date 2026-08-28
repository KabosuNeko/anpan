import {execFileSync} from 'node:child_process'

const CLIP_COMMANDS: ReadonlyArray<[string, string[]]> =
  process.platform === 'darwin'
    ? [['pbpaste', []]]
    : process.platform === 'win32'
      ? [['powershell', ['-NoProfile', '-Command', 'Get-Clipboard']]]
      : [
          ['wl-paste', ['--no-newline']],
          ['xclip', ['-selection', 'clipboard', '-o']],
          ['xsel', ['--clipboard', '--output']],
        ]

/** Read the system clipboard content. Returns empty string on failure. */
export function readClipboard(): string {
  for (const [cmd, args] of CLIP_COMMANDS) {
    try {
      return execFileSync(cmd, args, {encoding: 'utf8', timeout: 500, stdio: ['ignore', 'pipe', 'ignore']})
    } catch {
      // tool missing or clipboard empty — try next
    }
  }
  return ''
}
