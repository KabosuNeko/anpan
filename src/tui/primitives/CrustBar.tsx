import React from 'react'
import {Text} from 'ink'
import {useAnpanTheme} from '../theme/palette.js'

export function CrustBar({percent, width = 30}: {percent: number; width?: number}) {
  const palette = useAnpanTheme()
  const clamped = Math.max(0, Math.min(1, percent))
  const filled = Math.round(clamped * width)
  return (
    <Text>
      <Text color={palette.primary}>{'█'.repeat(filled)}</Text>
      <Text color={palette.muted} dimColor={palette.dimAccent}>{'░'.repeat(width - filled)}</Text>
      <Text color={palette.primary}> {`${Math.round(clamped * 100)}%`.padStart(4)}</Text>
    </Text>
  )
}
