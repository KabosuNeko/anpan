import {execFileSync} from 'node:child_process'

// Wayland (wl-paste) must precede X11 utilities on Linux: xclip under XWayland
// can hang or return stale X11 clipboard selections instead of native Wayland buffers.
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

export function readClipboard(): string {
  for (const [cmd, args] of CLIP_COMMANDS) {
    try {
      return execFileSync(cmd, args, {encoding: 'utf8', timeout: 500, stdio: ['ignore', 'pipe', 'ignore']})
    } catch {}
  }
  return ''
}
