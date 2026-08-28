/**
 * Click hit-testing against the rendered terminal frame.
 * Ink has no absolute-position API, so we keep the last frame ink wrote
 * and find clickable text by content matching.
 */

const ANSI_RE = new RegExp(
  [
    '[\\u001B\\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[-a-zA-Z\\d\\/#&.:=?%@~_]*)*)?\\u0007)',
    '(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PR-TZcf-nq-uy=><~]))',
  ].join('|'),
  'g',
)
const stripAnsi = (text: string) => text.replace(ANSI_RE, '')

let capturedLines: string[] = []

/**
 * Wrap the stdout stream so every frame ink writes is kept for hit-testing.
 * Cursor-only updates carry no printable text and are ignored.
 */
export function captureFrames<T extends NodeJS.WriteStream>(stream: T): T {
  return new Proxy(stream, {
    get(target, prop) {
      if (prop === 'write') {
        return (chunk: unknown, ...rest: unknown[]) => {
          const lines = String(chunk).split('\n').map(stripAnsi)
          if (lines.some(l => l.trim() !== '')) capturedLines = lines
          return (target.write as (...args: unknown[]) => boolean)(chunk, ...rest)
        }
      }
      const value = Reflect.get(target, prop)
      return typeof value === 'function' ? (value as (...a: unknown[]) => unknown).bind(target) : value
    },
  })
}

export type TapTarget = {
  /** Exact text to find in the frame (ANSI is stripped before matching). */
  match: string
  action: () => void
  /** Extra cells left/right of the match that still count as a hit. */
  padX?: number
  /** Extra rows above/below the match that still count as a hit. */
  padY?: number
}

/** First target whose text sits under the click. x/y are 1-based terminal cells. */
export function tapTargetAt(x: number, y: number, targets: TapTarget[]): TapTarget | undefined {
  for (const target of targets) {
    const {match, padX = 1, padY = 0} = target
    for (let row = y - 1 - padY; row <= y - 1 + padY; row++) {
      const line = capturedLines[row]
      if (!line) continue
      let idx = line.indexOf(match)
      while (idx !== -1) {
        if (x - 1 >= idx - padX && x - 1 <= idx + match.length - 1 + padX) return target
        idx = line.indexOf(match, idx + 1)
      }
    }
  }
  return undefined
}

/** Frame line index containing `text`, or -1. */
export function locateFrameRow(text: string): number {
  return capturedLines.findIndex(l => l.includes(text))
}

/** 1-based [first, last] columns of the visible text on a frame row. */
export function frameRowBounds(row: number): [number, number] | undefined {
  const line = capturedLines[row]
  if (!line) return undefined
  const first = line.search(/\S/)
  if (first === -1) return undefined
  return [first + 1, line.trimEnd().length]
}
