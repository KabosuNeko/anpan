import React from 'react'
import {Box, Text} from 'ink'
import {useAnpanTheme} from '../theme/palette.js'

export function FooterKeys({
  hints,
}: {
  hints: Array<[string, string]>
}) {
  const palette = useAnpanTheme()
  return (
    <Box gap={2}>
      {hints.map(([key, desc], i) => (
        <Text key={i} dimColor={palette.dimAccent}>
          <Text bold>{key}</Text> {desc}
        </Text>
      ))}
    </Box>
  )
}
