import React, {useState} from 'react'
import os from 'node:os'
import path from 'node:path'
import {Box, Text, useInput} from 'ink'
import {BunCard} from '../primitives/BunCard.js'
import {KeyField} from '../primitives/KeyField.js'
import {useAnpanTheme} from '../theme/palette.js'
import {shortenPath} from '../../core/units.js'
import type {AnpanConfig} from '../../system/config.js'

type SettingsViewProps = {
  width: number
  config: AnpanConfig
  onChange: (updated: AnpanConfig) => void
  onClose: () => void
}

type SettingKey =
  | 'aria2c'
  | 'connections'
  | 'askSaveDir'
  | 'outDir'
  | 'videoContainer'
  | 'audioFormat'
  | 'subtitles'
  | 'subLangs'
  | 'sponsorBlock'
  | 'cookiesBrowser'
  | 'embedMetadata'
  | 'writeThumbnail'
  | 'preferQuality'

const ITEMS: Array<{key: SettingKey; label: string}> = [
  {key: 'askSaveDir', label: 'ask save location'},
  {key: 'outDir', label: 'default save dir'},
  {key: 'videoContainer', label: 'video format (container)'},
  {key: 'audioFormat', label: 'audio format'},
  {key: 'subtitles', label: 'subtitles'},
  {key: 'subLangs', label: 'subtitle languages'},
  {key: 'sponsorBlock', label: 'sponsorblock'},
  {key: 'cookiesBrowser', label: 'browser cookies'},
  {key: 'aria2c', label: 'aria2c accelerator'},
  {key: 'connections', label: 'aria2c connections (-x -s)'},
  {key: 'embedMetadata', label: 'embed audio tags & cover'},
  {key: 'writeThumbnail', label: 'write thumbnail image'},
  {key: 'preferQuality', label: 'auto-select quality'},
]

const CONNECTION_CHOICES: ReadonlyArray<AnpanConfig['connections']> = [4, 8, 16, 32]
const QUALITY_CHOICES: ReadonlyArray<AnpanConfig['preferQuality']> = ['ask', 'best', '1080p', 'audio']
const CONTAINER_CHOICES: ReadonlyArray<AnpanConfig['videoContainer']> = ['mp4', 'mkv', 'webm']
const AUDIO_CHOICES: ReadonlyArray<AnpanConfig['audioFormat']> = ['mp3', 'm4a', 'opus', 'flac', 'wav']
const SUBTITLE_CHOICES: ReadonlyArray<AnpanConfig['subtitles']> = ['off', 'embed', 'write']
const SUBLANG_CHOICES: ReadonlyArray<AnpanConfig['subLangs']> = ['vi,en', 'all', 'en']
const SPONSORBLOCK_CHOICES: ReadonlyArray<AnpanConfig['sponsorBlock']> = ['off', 'remove', 'mark']
const COOKIES_CHOICES: ReadonlyArray<AnpanConfig['cookiesBrowser']> = [
  'none',
  'chrome',
  'firefox',
  'brave',
  'edge',
  'safari',
]

const DIR_PRESETS = [
  path.join(os.homedir(), 'Downloads'),
  path.join(os.homedir(), 'Videos'),
  path.join(os.homedir(), 'Desktop'),
  process.cwd(),
]

export function SettingsView({width, config, onChange, onClose}: SettingsViewProps) {
  const palette = useAnpanTheme()
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [editingDir, setEditingDir] = useState(false)
  const [dirInput, setDirInput] = useState('')

  const cycleValue = (key: SettingKey, direction: 1 | -1 = 1) => {
    if (key === 'aria2c') {
      onChange({...config, aria2c: !config.aria2c})
    } else if (key === 'askSaveDir') {
      onChange({...config, askSaveDir: !config.askSaveDir})
    } else if (key === 'embedMetadata') {
      onChange({...config, embedMetadata: !config.embedMetadata})
    } else if (key === 'writeThumbnail') {
      onChange({...config, writeThumbnail: !config.writeThumbnail})
    } else if (key === 'connections') {
      const idx = CONNECTION_CHOICES.indexOf(config.connections)
      const nextIdx = (idx + direction + CONNECTION_CHOICES.length) % CONNECTION_CHOICES.length
      onChange({...config, connections: CONNECTION_CHOICES[nextIdx]!})
    } else if (key === 'preferQuality') {
      const idx = QUALITY_CHOICES.indexOf(config.preferQuality)
      const nextIdx = (idx + direction + QUALITY_CHOICES.length) % QUALITY_CHOICES.length
      onChange({...config, preferQuality: QUALITY_CHOICES[nextIdx]!})
    } else if (key === 'videoContainer') {
      const idx = CONTAINER_CHOICES.indexOf(config.videoContainer)
      const nextIdx = (idx + direction + CONTAINER_CHOICES.length) % CONTAINER_CHOICES.length
      onChange({...config, videoContainer: CONTAINER_CHOICES[nextIdx]!})
    } else if (key === 'audioFormat') {
      const idx = AUDIO_CHOICES.indexOf(config.audioFormat)
      const nextIdx = (idx + direction + AUDIO_CHOICES.length) % AUDIO_CHOICES.length
      onChange({...config, audioFormat: AUDIO_CHOICES[nextIdx]!})
    } else if (key === 'subtitles') {
      const idx = SUBTITLE_CHOICES.indexOf(config.subtitles)
      const nextIdx = (idx + direction + SUBTITLE_CHOICES.length) % SUBTITLE_CHOICES.length
      onChange({...config, subtitles: SUBTITLE_CHOICES[nextIdx]!})
    } else if (key === 'subLangs') {
      const idx = SUBLANG_CHOICES.indexOf(config.subLangs)
      const nextIdx = (idx + direction + SUBLANG_CHOICES.length) % SUBLANG_CHOICES.length
      onChange({...config, subLangs: SUBLANG_CHOICES[nextIdx]!})
    } else if (key === 'sponsorBlock') {
      const idx = SPONSORBLOCK_CHOICES.indexOf(config.sponsorBlock)
      const nextIdx = (idx + direction + SPONSORBLOCK_CHOICES.length) % SPONSORBLOCK_CHOICES.length
      onChange({...config, sponsorBlock: SPONSORBLOCK_CHOICES[nextIdx]!})
    } else if (key === 'cookiesBrowser') {
      const idx = COOKIES_CHOICES.indexOf(config.cookiesBrowser)
      const nextIdx = (idx + direction + COOKIES_CHOICES.length) % COOKIES_CHOICES.length
      onChange({...config, cookiesBrowser: COOKIES_CHOICES[nextIdx]!})
    } else if (key === 'outDir') {
      const currentNorm = path.resolve(config.outDir)
      const presetNorms = DIR_PRESETS.map(p => path.resolve(p))
      let idx = presetNorms.indexOf(currentNorm)
      if (idx === -1) idx = 0
      const nextIdx = (idx + direction + DIR_PRESETS.length) % DIR_PRESETS.length
      onChange({...config, outDir: DIR_PRESETS[nextIdx]!})
    }
  }

  const saveCustomDir = (raw: string) => {
    let cleaned = raw.trim()
    if (!cleaned) return
    if (cleaned.startsWith('~')) {
      cleaned = path.join(os.homedir(), cleaned.slice(1))
    }
    const resolved = path.resolve(cleaned)
    onChange({...config, outDir: resolved})
    setEditingDir(false)
  }

  useInput((input, key) => {
    if (editingDir) {
      if (key.escape) {
        setEditingDir(false)
        return
      }
      return
    }

    if (key.escape || (key.ctrl && input === 's')) {
      onClose()
      return
    }

    if (key.upArrow || input === 'k') {
      setSelectedIndex(i => (i - 1 + ITEMS.length) % ITEMS.length)
      return
    }

    if (key.downArrow || input === 'j') {
      setSelectedIndex(i => (i + 1) % ITEMS.length)
      return
    }

    const current = ITEMS[selectedIndex]
    if (!current) return

    if (current.key === 'outDir') {
      if (key.return) {
        setDirInput(shortenPath(config.outDir, os.homedir()))
        setEditingDir(true)
        return
      }
      if (key.leftArrow || input === 'h') {
        cycleValue('outDir', -1)
        return
      }
      if (key.rightArrow || input === 'l' || input === ' ') {
        cycleValue('outDir', 1)
        return
      }
    } else {
      if (key.leftArrow || input === 'h') {
        cycleValue(current.key, -1)
        return
      }
      if (key.rightArrow || input === 'l' || key.return || input === ' ') {
        cycleValue(current.key, 1)
        return
      }
    }
  })

  const formatVal = (key: SettingKey) => {
    if (key === 'aria2c') return config.aria2c ? 'on' : 'off'
    if (key === 'askSaveDir') return config.askSaveDir ? 'always ask' : 'use default'
    if (key === 'embedMetadata') return config.embedMetadata ? 'on' : 'off'
    if (key === 'writeThumbnail') return config.writeThumbnail ? 'on' : 'off'
    if (key === 'connections') return `${config.connections}`
    if (key === 'videoContainer') return config.videoContainer
    if (key === 'audioFormat') return config.audioFormat
    if (key === 'subtitles') return config.subtitles
    if (key === 'subLangs') return config.subLangs
    if (key === 'sponsorBlock') return config.sponsorBlock
    if (key === 'cookiesBrowser') return config.cookiesBrowser
    if (key === 'preferQuality') return config.preferQuality
    if (key === 'outDir') return shortenPath(config.outDir, os.homedir(), 22)
    return ''
  }

  return (
    <BunCard title="settings" width={width}>
      <Box flexDirection="column" gap={0}>
        {ITEMS.map((item, index) => {
          const isSelected = index === selectedIndex
          const isCurrentEditing = isSelected && editingDir

          return (
            <Box key={item.key} justifyContent="space-between" width="100%">
              <Text>
                <Text dimColor={palette.dimAccent}>{isSelected ? '> ' : '  '}</Text>
                <Text bold={isSelected}>{item.label}</Text>
              </Text>
              {isCurrentEditing ? (
                <Box width={26} height={1}>
                  <Text dimColor={palette.dimAccent}>{'[ '}</Text>
                  <KeyField
                    value={dirInput}
                    onChange={setDirInput}
                    onSubmit={saveCustomDir}
                    width={22}
                  />
                  <Text dimColor={palette.dimAccent}>{' ]'}</Text>
                </Box>
              ) : (
                <Text bold={isSelected} dimColor={!isSelected && palette.dimAccent}>
                  {`[ ${formatVal(item.key)} ]`}
                </Text>
              )}
            </Box>
          )
        })}
      </Box>
    </BunCard>
  )
}
