import {useEffect, useRef} from 'react'
import {useStdin, useStdout} from 'ink'

const ENABLE = '\u001B[?1000h\u001B[?1006h'
const DISABLE = '\u001B[?1006l\u001B[?1000l'
const SGR_PRESS = /\u001B\[<(\d+);(\d+);(\d+)M/g

/**
 * Reports left-button presses as 1-based (column, row) terminal cells.
 * While active the terminal's native text selection needs a modifier key.
 */
export function usePointer(onClick: (x: number, y: number) => void, isActive: boolean) {
  const handlerRef = useRef(onClick)
  handlerRef.current = onClick
  const {stdin} = useStdin()
  const {stdout} = useStdout()

  useEffect(() => {
    if (!isActive || !stdin || !stdout || !process.stdin.isTTY) return
    stdout.write(ENABLE)
    const onData = (data: Buffer | string) => {
      for (const match of String(data).matchAll(SGR_PRESS)) {
        const [, button, x, y] = match
        if (button === '0') handlerRef.current(Number(x), Number(y))
      }
    }
    stdin.on('data', onData)
    return () => {
      stdin.off('data', onData)
      stdout.write(DISABLE)
    }
  }, [isActive, stdin, stdout])
}

/**
 * Mouse reports leak through ink's keypress parser as typed text.
 * Run every onChange value through this to drop leaked sequences.
 */
export const stripPointerReports = (value: string) =>
  value.replace(/\u001B?\[?<\d+;\d+;\d+[Mm]/g, '')
