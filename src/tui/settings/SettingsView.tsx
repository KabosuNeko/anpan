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

type SettingKey = 'aria2c' | 'connections' | 'preferQuality' | 'outDir'

const ITEMS: Array<{key: SettingKey; label: string}> = [
  {key: 'aria2c', label: 'aria2c accelerator'},
  {key: 'connections', label: 'aria2c connections (-x -s)'},
  {key: 'preferQuality', label: 'default format'},
  {key: 'outDir', label: 'save directory'},
]

const CONNECTION_CHOICES: ReadonlyArray<AnpanConfig['connections']> = [4, 8, 16, 32]
const QUALITY_CHOICES: ReadonlyArray<AnpanConfig['preferQuality']> = ['ask', 'best', '1080p', 'audio']

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
    } else if (key === 'connections') {
      const idx = CONNECTION_CHOICES.indexOf(config.connections)
      const nextIdx = (idx + direction + CONNECTION_CHOICES.length) % CONNECTION_CHOICES.length
      onChange({...config, connections: CONNECTION_CHOICES[nextIdx]!})
    } else if (key === 'preferQuality') {
      const idx = QUALITY_CHOICES.indexOf(config.preferQuality)
      const nextIdx = (idx + direction + QUALITY_CHOICES.length) % QUALITY_CHOICES.length
      onChange({...config, preferQuality: QUALITY_CHOICES[nextIdx]!})
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
    if (key === 'connections') return `${config.connections}`
    if (key === 'preferQuality') return config.preferQuality
    if (key === 'outDir') return shortenPath(config.outDir, os.homedir(), 24)
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
