import {type ReactNode} from 'react'
import {Box, Text} from 'ink'
import {useAnpanTheme} from '../theme/palette.js'
export function BunCard({title, width, children}: {title: string; width: number; children: ReactNode}) {
  const palette = useAnpanTheme()
  const inner = width - 2
  const tail = Math.max(0, inner - title.length - 3)
  return (
    <Box flexDirection="column" width={width}>
      <Text>
        <Text color={palette.muted} dimColor={palette.dimAccent}>{'╭─ '}</Text>
        <Text color={palette.primary}>{title}</Text>
        <Text color={palette.muted} dimColor={palette.dimAccent}>{` ${'─'.repeat(tail)}╮`}</Text>
      </Text>
      <Box
        width={width}
        borderStyle="round"
        borderColor={palette.muted}
        borderDimColor={palette.dimAccent}
        borderTop={false}
        flexDirection="column"
        paddingX={2}
      >
        {children}
      </Box>
    </Box>
  )
}
