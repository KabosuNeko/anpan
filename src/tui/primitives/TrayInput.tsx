import {type ReactNode} from 'react'
import {Box, Text} from 'ink'
import {useAnpanTheme} from '../theme/palette.js'

const buttonWidth = (label: string) => label.length + 4
export function TrayInput({
  title,
  width: totalWidth,
  actionLabel,
  actionDim = false,
  children,
}: {
  title: string
  width: number
  actionLabel?: string
  actionDim?: boolean
  children: ReactNode
}) {
  const palette = useAnpanTheme()

  const btnW = actionLabel ? buttonWidth(actionLabel) : 0
  const leftW = actionLabel ? totalWidth - btnW : totalWidth
  const leftInner = actionLabel
    ? Math.max(1, leftW - title.length - 4)
    : Math.max(1, totalWidth - title.length - 5)

  return (
    <Box flexDirection="column" width={totalWidth}>
      <Text>
        <Text dimColor={palette.dimAccent}>{'╭─ '}</Text><Text>{title}</Text><Text dimColor={palette.dimAccent}>{` ${'─'.repeat(leftInner)}`}</Text>{actionLabel ? <Text dimColor={palette.dimAccent}>{`┬${'─'.repeat(btnW - 2)}╮`}</Text> : <Text dimColor={palette.dimAccent}>{'╮'}</Text>}
      </Text>

      <Box width={totalWidth} height={1}>
        <Box width={leftW} height={1} overflow="hidden">
          <Text dimColor={palette.dimAccent}>{'│ > '}</Text>
          <Box flexGrow={1} height={1} overflow="hidden">
            {children}
          </Box>
        </Box>
        {actionLabel ? (
          <Box width={btnW} height={1}>
            <Text dimColor={palette.dimAccent}>{'│ '}</Text>
            <Text bold={!actionDim} dimColor={actionDim}>
              {actionLabel}
            </Text>
            <Text dimColor={palette.dimAccent}>{' │'}</Text>
          </Box>
        ) : (
          <Text dimColor={palette.dimAccent}>{' │'}</Text>
        )}
      </Box>

      <Text dimColor={palette.dimAccent}>
        {`╰${'─'.repeat(leftW - 1)}`}
        {actionLabel ? `┴${'─'.repeat(btnW - 2)}╯` : '╯'}
      </Text>
    </Box>
  )
}
