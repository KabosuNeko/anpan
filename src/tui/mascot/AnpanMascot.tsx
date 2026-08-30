import {useEffect, useMemo, useState} from 'react'
import {Box, Text} from 'ink'
import {type Palette, useAnpanTheme} from '../theme/palette.js'

export const MASCOT_MATCH = 'cdxkkx'

const ART = [
  '     .,cdxkkxoc,.             ',
  "  'cdkdxxdkkdxkkkd:.          ",
  '.dkkkkdxxdkdxkxxkkkko         ',
  'okkkkkkkkkkkkkkOKXXXXXK0Okc.  ',
  ' dkkkkkkkkkkx0WNNK000000KXNWk ',
  '   .kkkkkkkkOMKdc::::::::cd0Mx',
  "             'xc:::::::::::d; ",
]
const GRID = ART.map(line => [...line])
const ROWS = GRID.length

const INTRO_MS = 800
const INTRO_SPREAD_MS = 450
const SWEEP_MS = 1000
const SWEEP_EVERY_MS = 7_000
const TILT = 1.5
const HALF = 3.0

const ease = (t: number) => 1 - Math.pow(1 - t, 3)

type AnimPhase = 'intro' | 'idle' | 'shimmer'

function cellAt(ch: string, row: number, col: number, phase: AnimPhase, t: number, delay: number, palette: Palette) {
  if (ch === ' ' || phase === 'idle') return {ch, color: palette.primary, dim: false}
  if (phase === 'intro') {
    const dt = t - delay
    if (dt < 0) return {ch: ' ', color: palette.primary, dim: false}
    if (dt < 100) return {ch: '.', color: palette.muted, dim: palette.dimAccent}
    if (dt < 200) return {ch: ':', color: palette.muted, dim: palette.dimAccent}
    return {ch, color: palette.primary, dim: false}
  }

  const cols = GRID[0]!.length
  const pMin = -TILT * ROWS - HALF
  const pMax = cols + HALF
  const p = pMin + ease(t / SWEEP_MS) * (pMax - pMin)
  const d = Math.abs(col - (ROWS - 1 - row) * TILT - p)
  if (d <= HALF && 1 - d / HALF > 0.3) {
    return {ch, color: palette.muted, dim: true}
  }
  return {ch, color: palette.primary, dim: false}
}

function renderRow(row: number, phase: AnimPhase, t: number, delays: number[], palette: Palette) {
  const segments: Array<{text: string; color?: string; dim: boolean}> = []
  GRID[row]!.forEach((ch, col) => {
    const cell = cellAt(ch, row, col, phase, t, delays[col]!, palette)
    const last = segments[segments.length - 1]
    if (last && ((last.color === cell.color && last.dim === cell.dim) || cell.ch === ' ')) {
      last.text += cell.ch
    } else {
      segments.push({text: cell.ch, color: cell.color, dim: cell.dim})
    }
  })
  return segments.map((seg, i) => (
    <Text key={i} color={seg.color} dimColor={seg.dim}>
      {seg.text}
    </Text>
  ))
}

export function AnpanMascot() {
  const palette = useAnpanTheme()
  const animated = Boolean(process.stdout.isTTY)
  const delays = useMemo(
    () => GRID.map(row => row.map(() => Math.random() * INTRO_SPREAD_MS)),
    [],
  )
  const [phase, setPhase] = useState<AnimPhase>(animated ? 'intro' : 'idle')
  const [t, setT] = useState(0)

  useEffect(() => {
    if (!animated) return
    if (phase === 'idle') {
      const id = setTimeout(() => {
        setT(0)
        setPhase('shimmer')
      }, SWEEP_EVERY_MS)
      return () => clearTimeout(id)
    }
    const duration = phase === 'intro' ? INTRO_MS : SWEEP_MS
    const start = Date.now()
    const id = setInterval(() => {
      const elapsed = Date.now() - start
      if (elapsed >= duration) {
        setT(0)
        setPhase('idle')
      } else {
        setT(elapsed)
      }
    }, 33)
    return () => clearInterval(id)
  }, [phase, animated])

  return (
    <Box flexDirection="column" flexShrink={0}>
      {GRID.map((_, row) => (
        <Text key={row}>{renderRow(row, phase, t, delays[row]!, palette)}</Text>
      ))}
    </Box>
  )
}
