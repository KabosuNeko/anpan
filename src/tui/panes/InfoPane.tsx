import React from 'react'
import os from 'node:os'
import {Box, Text} from 'ink'
import {useAnpanTheme} from '../theme/palette.js'
import {formatDuration, shortenPath, truncate} from '../../core/units.js'
import type {SiteInfo} from '../../core/domains.js'
import type {VideoMeta} from '../../engine/extractor.js'
import type {AnpanConfig} from '../../system/config.js'

type InfoPaneProps = {
  width: number
  stageName: string
  meta: VideoMeta | null
  platform: SiteInfo | null
  config: AnpanConfig
  hasAria2c: boolean
}

export function InfoPane({width, stageName, meta, platform, config, hasAria2c}: InfoPaneProps) {
  const palette = useAnpanTheme()
  const contentWidth = width - 4

  const engineLabel = config.aria2c && hasAria2c
    ? `aria2c (${config.connections})`
    : 'native (yt-dlp)'

  return (
    <Box
      flexDirection="column"
      width={width}
      borderStyle="round"
      borderColor={palette.muted}
      borderDimColor={palette.dimAccent}
      paddingX={1}
    >
      <Box justifyContent="space-between">
        <Text bold>anpan</Text>
        <Text dimColor={palette.dimAccent}>v0.1.0</Text>
      </Box>

      <Text dimColor={palette.dimAccent}>{'─'.repeat(contentWidth)}</Text>

      {/* Video Details if Probed */}
      {meta ? (
        <Box flexDirection="column" gap={0}>
          <Box justifyContent="space-between">
            <Text dimColor={palette.dimAccent}>site</Text>
            <Text bold>{truncate(platform?.label ?? 'unknown', contentWidth - 6)}</Text>
          </Box>
          {meta.duration ? (
            <Box justifyContent="space-between">
              <Text dimColor={palette.dimAccent}>length</Text>
              <Text>{formatDuration(meta.duration)}</Text>
            </Box>
          ) : null}
          <Box flexDirection="column">
            <Text dimColor={palette.dimAccent}>title</Text>
            <Text bold>{`  ${truncate(meta.title, contentWidth - 2)}`}</Text>
          </Box>
          {meta.uploader ? (
            <Box flexDirection="column">
              <Text dimColor={palette.dimAccent}>channel</Text>
              <Text>{`  ${truncate(meta.uploader, contentWidth - 2)}`}</Text>
            </Box>
          ) : null}
        </Box>
      ) : (
        <Box flexDirection="column" gap={0}>
          <Box justifyContent="space-between">
            <Text dimColor={palette.dimAccent}>status</Text>
            <Text>{stageName}</Text>
          </Box>
          <Box justifyContent="space-between">
            <Text dimColor={palette.dimAccent}>format</Text>
            <Text>{config.preferQuality}</Text>
          </Box>
        </Box>
      )}

      <Text dimColor={palette.dimAccent}>{'─'.repeat(contentWidth)}</Text>

      {/* System / Engine Details */}
      <Box flexDirection="column" gap={0}>
        <Box justifyContent="space-between">
          <Text dimColor={palette.dimAccent}>engine</Text>
          <Text>{engineLabel}</Text>
        </Box>
        <Box justifyContent="space-between">
          <Text dimColor={palette.dimAccent}>out</Text>
          <Text>{shortenPath(config.outDir, os.homedir(), contentWidth - 5)}</Text>
        </Box>
      </Box>
    </Box>
  )
}
