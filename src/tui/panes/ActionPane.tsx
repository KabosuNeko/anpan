import React from 'react'
import os from 'node:os'
import {Box, Text} from 'ink'
import SelectInput, {type IndicatorProps, type ItemProps} from 'ink-select-input'
import Spinner from 'ink-spinner'
import {CrustBar} from '../primitives/CrustBar.js'
import {KeyField} from '../primitives/KeyField.js'
import {SettingsView} from '../settings/SettingsView.js'
import {useAnpanTheme} from '../theme/palette.js'
import {formatBytes, formatEta, formatSpeed, shortenPath, wrapText} from '../../core/units.js'
import type {Portion} from '../../engine/extractor.js'
import type {BakeProgress} from '../../engine/downloader.js'
import type {AnpanConfig} from '../../system/config.js'

type ActionPaneProps = {
  width: number
  stage:
    | {name: 'input'; warning?: string}
    | {name: 'probing'; status: string}
    | {name: 'selecting'}
    | {name: 'baking'; portion: Portion; progress?: BakeProgress; processing: boolean}
    | {name: 'baked'; filepath: string}
    | {name: 'error'; message: string}
  url: string
  onUrlChange: (url: string) => void
  onUrlSubmit: (url: string) => void
  clipboardUrl?: string
  history: string[]
  portions: Portion[]
  onPortionSelect: (portion: Portion) => void
  showSettings: boolean
  config: AnpanConfig
  onConfigChange: (updated: AnpanConfig) => void
  onCloseSettings: () => void
  onReset: () => void
}

function PortionIndicator({isSelected}: IndicatorProps) {
  const palette = useAnpanTheme()
  return (
    <Box marginRight={1}>
      <Text dimColor={!isSelected && palette.dimAccent}>{isSelected ? '❯' : ' '}</Text>
    </Box>
  )
}

function PortionItem({isSelected, label}: ItemProps) {
  return <Text bold={isSelected}>{label}</Text>
}

export function ActionPane({
  width,
  stage,
  url,
  onUrlChange,
  onUrlSubmit,
  clipboardUrl,
  history,
  portions,
  onPortionSelect,
  showSettings,
  config,
  onConfigChange,
  onCloseSettings,
  onReset,
}: ActionPaneProps) {
  const palette = useAnpanTheme()
  const innerWidth = width - 4

  const title = showSettings
    ? 'settings'
    : stage.name === 'input'
      ? 'url'
      : stage.name === 'probing'
        ? 'probing'
        : stage.name === 'selecting'
          ? 'format'
          : stage.name === 'baking'
            ? 'download'
            : stage.name === 'baked'
              ? 'done'
              : 'error'

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
        <Text bold>{title}</Text>
        <Text dimColor={palette.dimAccent}>action</Text>
      </Box>

      <Text dimColor={palette.dimAccent}>{'─'.repeat(innerWidth)}</Text>

      {/* ── Settings View ─────────────────────────────────── */}
      {showSettings ? (
        <Box flexDirection="column">
          <SettingsView
            width={innerWidth}
            config={config}
            onChange={onConfigChange}
            onClose={onCloseSettings}
            bare
          />
        </Box>
      ) : (
        <>
          {/* ── Input Stage ───────────────────────────────── */}
          {stage.name === 'input' && (
            <Box flexDirection="column" gap={1}>
              <Box width={innerWidth} height={1}>
                <Text dimColor={palette.dimAccent}>{'> '}</Text>
                <KeyField
                  value={url}
                  onChange={onUrlChange}
                  onSubmit={onUrlSubmit}
                  placeholder={clipboardUrl ? `${clipboardUrl}  ⇥ paste` : 'https://...'}
                  width={innerWidth - 2}
                  history={history}
                  submitOnPaste={u => Boolean(u.trim())}
                  onTab={clipboardUrl ? () => {onUrlChange(clipboardUrl); onUrlSubmit(clipboardUrl)} : undefined}
                />
              </Box>
              <Box justifyContent="flex-end">
                <Text bold={Boolean(url.trim())} dimColor={!url.trim()}>
                  {'[ bake ]'}
                </Text>
              </Box>
              {stage.warning && <Text color="yellow">{stage.warning}</Text>}
            </Box>
          )}

          {/* ── Probing Stage ─────────────────────────────── */}
          {stage.name === 'probing' && (
            <Box gap={1} alignItems="center">
              <Text>
                <Spinner type="dots" />
              </Text>
              <Text dimColor={palette.dimAccent}>{stage.status}</Text>
            </Box>
          )}

          {/* ── Selecting Stage ───────────────────────────── */}
          {stage.name === 'selecting' && (
            <Box flexDirection="column">
              <SelectInput
                items={portions.map(p => ({label: p.label, value: p}))}
                onSelect={item => onPortionSelect(item.value)}
                indicatorComponent={PortionIndicator}
                itemComponent={PortionItem}
              />
            </Box>
          )}

          {/* ── Baking / Download Stage ───────────────────── */}
          {stage.name === 'baking' && (
            <Box flexDirection="column" gap={1}>
              {stage.processing ? (
                <Box gap={1} alignItems="center">
                  <Text>
                    <Spinner type="dots" />
                  </Text>
                  <Text dimColor={palette.dimAccent}>processing / merging…</Text>
                </Box>
              ) : stage.progress ? (
                <Box flexDirection="column" gap={0}>
                  {stage.progress.totalParts > 1 ? (
                    <Text dimColor={palette.dimAccent}>
                      {`part ${stage.progress.part + 1}/${stage.progress.totalParts}`}
                    </Text>
                  ) : null}
                  {stage.progress.totalBytes ? (
                    <CrustBar
                      percent={stage.progress.downloadedBytes / stage.progress.totalBytes}
                      width={Math.min(30, innerWidth - 8)}
                    />
                  ) : (
                    <Text dimColor={palette.dimAccent}>
                      {formatBytes(stage.progress.downloadedBytes)}
                    </Text>
                  )}
                  <Box justifyContent="space-between">
                    <Text dimColor={palette.dimAccent}>
                      {stage.progress.speed ? `${formatSpeed(stage.progress.speed)}` : ''}
                    </Text>
                    <Text dimColor={palette.dimAccent}>
                      {stage.progress.eta ? `${formatEta(stage.progress.eta)} left` : ''}
                    </Text>
                  </Box>
                </Box>
              ) : (
                <Box gap={1} alignItems="center">
                  <Text>
                    <Spinner type="dots" />
                  </Text>
                  <Text dimColor={palette.dimAccent}>downloading…</Text>
                </Box>
              )}
            </Box>
          )}

          {/* ── Baked / Completed Stage ───────────────────── */}
          {stage.name === 'baked' && (
            <Box flexDirection="column" gap={1}>
              <Text bold>downloaded</Text>
              <Text dimColor={palette.dimAccent}>
                {shortenPath(stage.filepath, os.homedir(), innerWidth)}
              </Text>
              <Text dimColor={palette.dimAccent}>↵ download another</Text>
            </Box>
          )}

          {/* ── Error Stage ───────────────────────────────── */}
          {stage.name === 'error' && (
            <Box flexDirection="column" gap={1}>
              <Text color="red" bold>download failed</Text>
              {wrapText(stage.message, innerWidth).map((line, i) => (
                <Text key={i} dimColor={palette.dimAccent}>{line}</Text>
              ))}
              <Text dimColor={palette.dimAccent}>↵ retry</Text>
            </Box>
          )}
        </>
      )}
    </Box>
  )
}
