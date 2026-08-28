import React, {useCallback, useEffect, useRef, useState} from 'react'
import os from 'node:os'
import {Box, Text, useApp, useInput, useStdout} from 'ink'
import SelectInput, {type IndicatorProps, type ItemProps} from 'ink-select-input'
import Spinner from 'ink-spinner'
import {BunCard} from './primitives/BunCard.js'
import {CrustBar} from './primitives/CrustBar.js'
import {FooterKeys} from './primitives/FooterKeys.js'
import {KeyField} from './primitives/KeyField.js'
import {StageViewport} from './primitives/StageViewport.js'
import {TrayInput} from './primitives/TrayInput.js'
import {AnpanMascot} from './mascot/AnpanMascot.js'
import {SettingsView} from './settings/SettingsView.js'
import {tapTargetAt, locateFrameRow, frameRowBounds, type TapTarget} from './events/hitTest.js'
import {usePointer} from './events/usePointer.js'
import {ThemeProvider, useAnpanTheme} from './theme/palette.js'
import {formatBytes, formatDuration, formatEta, formatSpeed, shortenPath, truncate, wrapText} from '../core/units.js'
import {addToHistory, loadHistory} from '../system/history.js'
import {loadConfig, saveConfig, type AnpanConfig} from '../system/config.js'
import {identifySite, isLikelyUrl, type SiteInfo} from '../core/domains.js'
import {
  extractPortions,
  probeVideo,
  type Portion,
  type VideoMeta,
} from '../engine/extractor.js'
import {bakeVideo, type BakeProgress} from '../engine/downloader.js'
import {ensureYtDlpBinary} from '../engine/binary.js'
import {findFfmpeg} from '../engine/ffmpeg.js'
import {findAria2c, buildAria2cArgs} from '../engine/aria2c.js'

export type {BakeProgress}

const BAKE_BUTTON = 'bake'
const DONE_LABEL = '↵ download another'
const TAGLINE = 'minimal terminal video downloader'

const portionLabel = (p: Portion) => p.label

function PortionIndicator({isSelected}: IndicatorProps) {
  const palette = useAnpanTheme()
  return (
    <Box marginRight={1}>
      <Text dimColor={!isSelected && palette.dimAccent}>{isSelected ? '❯' : ' '}</Text>
    </Box>
  )
}

function PortionItem({isSelected, label}: ItemProps) {
  return (
    <Text bold={isSelected}>
      {label}
    </Text>
  )
}

const Gap = ({lines = 1}: {lines?: number}) => (
  <Box flexDirection="column" flexShrink={0}>
    {Array.from({length: lines}, (_, i) => (
      <Text key={i}> </Text>
    ))}
  </Box>
)

function partTag(progress: BakeProgress): string {
  return progress.totalParts > 1 ? `part ${progress.part + 1}/${progress.totalParts}  ` : ''
}

function progressMeta(progress: BakeProgress): string {
  const speed = progress.speed ? formatSpeed(progress.speed) : ''
  const eta = progress.eta ? `${formatEta(progress.eta)} left` : ''
  return `${partTag(progress)}${speed.padStart(10)}  ${eta.padEnd(12)}`
}

function indeterminateMeta(progress: BakeProgress): string {
  const bytes = formatBytes(progress.downloadedBytes)
  const speed = progress.speed ? formatSpeed(progress.speed) : ''
  return `${partTag(progress)}${bytes.padStart(8)}  ${speed.padEnd(10)}`
}

export type Outcome = {filepath?: string}

type Stage =
  | {name: 'input'; warning?: string}
  | {name: 'probing'; status: string}
  | {name: 'selecting'}
  | {
      name: 'baking'
      portion: Portion
      progress?: BakeProgress
      processing: boolean
    }
  | {name: 'baked'; filepath: string}
  | {name: 'error'; message: string}

const HINTS: Record<Stage['name'], Array<[string, string]>> = {
  input: [
    ['↵', 'bake'],
    ['^s', 'settings'],
    ['^c', 'quit'],
  ],
  probing: [
    ['esc', 'cancel'],
    ['^c', 'quit'],
  ],
  selecting: [
    ['↑↓', 'choose'],
    ['↵', 'download'],
    ['esc', 'back'],
    ['^c', 'quit'],
  ],
  baking: [
    ['esc', 'cancel'],
    ['^c', 'quit'],
  ],
  baked: [
    ['↵', 'again'],
    ['^s', 'settings'],
    ['^c', 'quit'],
  ],
  error: [
    ['↵', 'retry'],
    ['^c', 'quit'],
  ],
}

type AnpanAppProps = {
  initialUrl?: string
  clipboardUrl?: string
  onOutcome: (outcome: Outcome) => void
}

export function AnpanApp(props: AnpanAppProps) {
  return (
    <ThemeProvider>
      <AppContent {...props} />
    </ThemeProvider>
  )
}

function AppContent({
  initialUrl,
  clipboardUrl,
  onOutcome,
}: AnpanAppProps) {
  const palette = useAnpanTheme()
  const {exit} = useApp()
  const {stdout} = useStdout()
  const cols = stdout?.columns ?? 80

  const [config, setConfig] = useState(loadConfig)
  const [showSettings, setShowSettings] = useState(false)

  const [url, setUrl] = useState(initialUrl ?? '')
  const [stage, setStage] = useState<Stage>(
    initialUrl
      ? {name: 'probing', status: 'probing video…'}
      : {name: 'input'},
  )
  const [history, setHistory] = useState(loadHistory)
  const [meta, setMeta] = useState<VideoMeta | null>(null)
  const [portions, setPortions] = useState<Portion[]>([])
  const [platform, setPlatform] = useState<SiteInfo | null>(null)
  const [cachedJsonPath, setCachedJsonPath] = useState<string | undefined>()

  const abortRef = useRef<AbortController | null>(null)
  const ytdlpRef = useRef<string>('')
  const ffmpegRef = useRef<string | undefined>(undefined)
  const aria2cPathRef = useRef<string | undefined>(undefined)

  const cancel = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
  }, [])

  const startBake = useCallback(
    async (portion: Portion) => {
      cancel()
      setStage({name: 'baking', portion, processing: false})

      const ac = new AbortController()
      abortRef.current = ac

      try {
        const aria2cArgs = config.aria2c
          ? buildAria2cArgs(aria2cPathRef.current, config.connections)
          : []

        const filepath = await bakeVideo(
          {
            ytdlpBin: ytdlpRef.current,
            ffmpegLocation: ffmpegRef.current,
            aria2cArgs,
            url,
            cachedJsonPath,
            portion,
            outputDir: config.outDir,
          },
          {
            onProgress: progress =>
              setStage(s => (s.name === 'baking' ? {...s, progress} : s)),
            onProcessing: () =>
              setStage(s => (s.name === 'baking' ? {...s, processing: true} : s)),
          },
          ac.signal,
        )
        onOutcome({filepath})
        setStage({name: 'baked', filepath})
      } catch (err) {
        if (ac.signal.aborted) {
          setStage({name: 'selecting'})
        } else {
          setStage({name: 'error', message: (err as Error).message})
        }
      }
    },
    [cancel, url, cachedJsonPath, config, onOutcome],
  )

  const startProbe = useCallback(
    async (targetUrl: string) => {
      cancel()
      setUrl(targetUrl)
      setPlatform(identifySite(targetUrl))
      setStage({name: 'probing', status: 'probing video…'})

      const ac = new AbortController()
      abortRef.current = ac

      try {
        const bin = await ensureYtDlpBinary(
          msg => setStage(s => (s.name === 'probing' ? {...s, status: msg} : s)),
          ac.signal,
        )
        ytdlpRef.current = bin

        const [ffmpeg, aria2c] = await Promise.all([findFfmpeg(), findAria2c()])
        ffmpegRef.current = ffmpeg
        aria2cPathRef.current = aria2c

        const result = await probeVideo(bin, targetUrl, ac.signal)
        setMeta(result.meta)
        setCachedJsonPath(result.cachedJsonPath)
        const resolvedPortions = extractPortions(result.meta)
        setPortions(resolvedPortions)
        setHistory(addToHistory(targetUrl))

        // auto-select if configured
        if (config.preferQuality === 'best' && resolvedPortions[0]) {
          void startBake(resolvedPortions[0])
        } else if (config.preferQuality === 'audio') {
          const audioChoice = resolvedPortions.find(p => p.kind === 'audio')
          if (audioChoice) void startBake(audioChoice)
          else setStage({name: 'selecting'})
        } else if (config.preferQuality === '1080p') {
          const p1080 = resolvedPortions.find(p => p.label.startsWith('1080p'))
          if (p1080) void startBake(p1080)
          else setStage({name: 'selecting'})
        } else {
          setStage({name: 'selecting'})
        }
      } catch (err) {
        if (ac.signal.aborted) {
          setStage({name: 'input'})
        } else {
          setStage({name: 'error', message: (err as Error).message})
        }
      }
    },
    [cancel, config, startBake],
  )

  useEffect(() => {
    if (initialUrl) void startProbe(initialUrl)
  }, [])

  useInput((input, key) => {
    if (key.ctrl && input === 'c') {
      cancel()
      exit()
      return
    }

    if (key.ctrl && input === 's' && stage.name !== 'probing' && stage.name !== 'baking') {
      setShowSettings(s => !s)
      return
    }

    if (showSettings) {
      // handled by SettingsView
      return
    }

    if (key.escape) {
      if (stage.name === 'probing' || stage.name === 'baking') {
        cancel()
      } else if (stage.name === 'selecting') {
        setStage({name: 'input'})
      }
    }
    if (key.return && stage.name === 'error') {
      setStage({name: 'input'})
    }
    if (key.return && stage.name === 'baked') {
      setUrl('')
      setMeta(null)
      setPortions([])
      setPlatform(null)
      setCachedJsonPath(undefined)
      setStage({name: 'input'})
    }
  })

  // mouse clicks
  const clickTargets = useRef<TapTarget[]>([])
  usePointer(
    (x, y) => {
      const target = tapTargetAt(x, y, clickTargets.current)
      target?.action()
    },
    stage.name !== 'probing' && !showSettings,
  )

  useEffect(() => {
    const targets: TapTarget[] = []
    const mascotRow = locateFrameRow('█▀█')
    if (mascotRow !== -1) {
      const bounds = frameRowBounds(mascotRow)
      if (bounds) {
        targets.push({
          match: '█▀█',
          padY: 2,
          action: () => {
            cancel()
            setShowSettings(false)
            setUrl('')
            setMeta(null)
            setStage({name: 'input'})
          },
        })
      }
    }

    if (stage.name === 'input') {
      targets.push({
        match: BAKE_BUTTON,
        padY: 1,
        action: () => {
          if (url.trim() && isLikelyUrl(url.trim())) void startProbe(url.trim())
        },
      })
    }

    if (stage.name === 'baked') {
      targets.push({
        match: DONE_LABEL,
        action: () => {
          setUrl('')
          setMeta(null)
          setStage({name: 'input'})
        },
      })
    }

    targets.push({
      match: 'settings',
      action: () => setShowSettings(s => !s),
    })

    clickTargets.current = targets
  }, [stage, url, showSettings])

  const panelWidth = Math.min(64, cols - 4)

  return (
    <StageViewport>
      <AnpanMascot />
      <Gap />
      <Text dimColor={palette.dimAccent}>
        {TAGLINE}
      </Text>
      <Gap />

      {/* ── Settings View ─────────────────────────────────────── */}
      {showSettings ? (
        <>
          <SettingsView
            width={panelWidth}
            config={config}
            onChange={updated => {
              setConfig(updated)
              saveConfig(updated)
            }}
            onClose={() => setShowSettings(false)}
          />
          <Gap />
          <FooterKeys hints={[['↑↓', 'select'], ['↵', 'edit/toggle'], ['⇄', 'preset'], ['esc', 'close']]} />
        </>
      ) : (
        <>
          {/* ── Input Stage ───────────────────────────────────── */}
          {stage.name === 'input' && (
            <>
              <TrayInput
                title="url"
                width={panelWidth}
                actionLabel={BAKE_BUTTON}
                actionDim={!url.trim()}
              >
                <KeyField
                  value={url}
                  onChange={setUrl}
                  onSubmit={v => {
                    const trimmed = v.trim()
                    if (trimmed && isLikelyUrl(trimmed)) void startProbe(trimmed)
                  }}
                  placeholder={clipboardUrl ? `${clipboardUrl}  ⇥ paste` : 'https://...'}
                  width={panelWidth - 8}
                  history={history}
                  submitOnPaste={isLikelyUrl}
                  onTab={clipboardUrl ? () => {setUrl(clipboardUrl); void startProbe(clipboardUrl)} : undefined}
                />
              </TrayInput>
              {stage.warning && (
                <>
                  <Gap />
                  <Text color="yellow">{stage.warning}</Text>
                </>
              )}
            </>
          )}

          {/* ── Probing Stage ─────────────────────────────────── */}
          {stage.name === 'probing' && (
            <Box>
              <Text>
                <Spinner type="dots" />
              </Text>
              <Text dimColor={palette.dimAccent}>
                {' '}
                {stage.status}
              </Text>
            </Box>
          )}

          {/* ── Selecting Stage ───────────────────────────────── */}
          {stage.name === 'selecting' && meta && (
            <>
              <Box flexDirection="column" alignItems="center" width={panelWidth}>
                <Text bold>
                  {truncate(meta.title, panelWidth)}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  {[meta.uploader, meta.duration ? formatDuration(meta.duration) : '', platform?.label]
                    .filter(Boolean)
                    .join(' · ')}
                </Text>
              </Box>
              <Gap />
              <BunCard title="format" width={panelWidth}>
                <SelectInput
                  items={portions.map(p => ({label: portionLabel(p), value: p}))}
                  onSelect={item => void startBake(item.value)}
                  indicatorComponent={PortionIndicator}
                  itemComponent={PortionItem}
                />
              </BunCard>
            </>
          )}

          {/* ── Baking / Downloading Stage ────────────────────── */}
          {stage.name === 'baking' && (
            <>
              {meta && (
                <Text bold>
                  {truncate(meta.title, panelWidth)}
                </Text>
              )}
              <Gap />
              {stage.processing ? (
                <Box>
                  <Text>
                    <Spinner type="dots" />
                  </Text>
                  <Text dimColor={palette.dimAccent}>
                    {' '}
                    processing / merging…
                  </Text>
                </Box>
              ) : stage.progress ? (
                <>
                  {stage.progress.totalBytes ? (
                    <>
                      <CrustBar percent={stage.progress.downloadedBytes / stage.progress.totalBytes} width={Math.min(40, panelWidth - 10)} />
                      <Text dimColor={palette.dimAccent}>
                        {progressMeta(stage.progress)}
                      </Text>
                    </>
                  ) : (
                    <Text dimColor={palette.dimAccent}>
                      {indeterminateMeta(stage.progress)}
                    </Text>
                  )}
                </>
              ) : (
                <Box>
                  <Text>
                    <Spinner type="dots" />
                  </Text>
                  <Text dimColor={palette.dimAccent}>
                    {' '}
                    downloading…
                  </Text>
                </Box>
              )}
            </>
          )}

          {/* ── Baked / Completed Stage ───────────────────────── */}
          {stage.name === 'baked' && (
            <>
              <Text bold>
                downloaded
              </Text>
              <Text dimColor={palette.dimAccent}>
                {shortenPath(stage.filepath, os.homedir(), panelWidth)}
              </Text>
              <Gap />
              <Text>{DONE_LABEL}</Text>
            </>
          )}

          {/* ── Error Stage ───────────────────────────────────── */}
          {stage.name === 'error' && (
            <>
              <Text color="red" bold>
                download failed
              </Text>
              {wrapText(stage.message, panelWidth).map((line, i) => (
                <Text key={i} dimColor={palette.dimAccent}>
                  {line}
                </Text>
              ))}
              <Gap />
              <Text>↵ retry</Text>
            </>
          )}

          <Gap />
          <FooterKeys hints={HINTS[stage.name]} />
        </>
      )}
    </StageViewport>
  )
}
