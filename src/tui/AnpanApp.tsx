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
import {AnpanMascot, MASCOT_MATCH} from './mascot/AnpanMascot.js'
import {SettingsView} from './settings/SettingsView.js'
import {tapTargetAt, locateFrameRow, frameRowBounds, type TapTarget} from './events/hitTest.js'
import {usePointer} from './events/usePointer.js'
import {ThemeProvider, useAnpanTheme} from './theme/palette.js'
import {formatBytes, formatDuration, formatEta, formatSpeed, shortenPath, truncate, wrapText} from '../core/units.js'
import {addToHistory, loadHistory} from '../system/history.js'
import {loadConfig, saveConfig, type AnpanConfig} from '../system/config.js'
import {identifySite, isLikelyTarget, isPlaylistUrl, type SiteInfo} from '../core/domains.js'
import {inspectTarget} from '../core/router.js'
import {
  extractPlaylistPortions,
  extractPortions,
  probePlaylist,
  probeVideo,
  type PlaylistMeta,
  type Portion,
  type VideoMeta,
} from '../engine/extractor.js'
import {bakeVideo, type BakeProgress} from '../engine/downloader.js'
import {ensureYtDlpBinary} from '../engine/binary.js'
import {findFfmpeg} from '../engine/ffmpeg.js'
import {findAria2c, buildAria2cArgs, bakeDirectDownload, bakeTorrentDownload} from '../engine/aria2c.js'

const BAKE_BUTTON = 'bake'
const DONE_LABEL = '↵ download another'
const TAGLINE = 'feed a link, bake a file.'
const SUPPORTED_HINT = 'youtube · x · instagram · soundcloud · torrent · and more'



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
  let tag = ''
  if (progress.playlistItem && progress.playlistTotal) {
    tag += `[${progress.playlistItem}/${progress.playlistTotal}] `
  }
  if (progress.totalParts > 1) {
    tag += `part ${progress.part + 1}/${progress.totalParts}  `
  }
  return tag
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
  | {name: 'playlist_prompt'; playlist: PlaylistMeta}
  | {name: 'direct_prompt'; filename: string; size?: number; url: string}
  | {name: 'torrent_prompt'; title: string; target: string}
  | {name: 'selecting'}
  | {
      name: 'baking'
      portion?: Portion
      targetTitle?: string
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
  playlist_prompt: [
    ['↑↓', 'choose'],
    ['↵', 'select'],
    ['esc', 'cancel'],
    ['^c', 'quit'],
  ],
  direct_prompt: [
    ['↑↓', 'choose'],
    ['↵', 'download'],
    ['esc', 'cancel'],
    ['^c', 'quit'],
  ],
  torrent_prompt: [
    ['↑↓', 'choose'],
    ['↵', 'download'],
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
  const [timeRange, setTimeRange] = useState<string | undefined>()
  const [timeLabel, setTimeLabel] = useState<string | undefined>()
  const [isPlaylistMode, setIsPlaylistMode] = useState(false)
  const [playlistMeta, setPlaylistMeta] = useState<PlaylistMeta | null>(null)

  const [stage, setStage] = useState<Stage>(
    initialUrl
      ? {name: 'probing', status: 'probing target…'}
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

  const resetToInput = useCallback(() => {
    setUrl('')
    setMeta(null)
    setPortions([])
    setPlatform(null)
    setCachedJsonPath(undefined)
    setTimeRange(undefined)
    setTimeLabel(undefined)
    setIsPlaylistMode(false)
    setPlaylistMeta(null)
    setStage({name: 'input'})
  }, [])

  const handleUrlChange = useCallback((newUrl: string) => {
    setUrl(newUrl)
    if (!newUrl.trim()) {
      resetToInput()
    }
  }, [resetToInput])

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
            timeRange,
            isPlaylist: isPlaylistMode,
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
    [cancel, url, cachedJsonPath, config, timeRange, isPlaylistMode, onOutcome],
  )

  const startDirectBake = useCallback(
    async (targetUrl: string, filename: string) => {
      cancel()
      setStage({name: 'baking', targetTitle: filename, processing: false})

      const ac = new AbortController()
      abortRef.current = ac

      try {
        const bin = aria2cPathRef.current || (await findAria2c())
        if (!bin) throw new Error('aria2c is required for direct multi-connection downloads.')

        const filepath = await bakeDirectDownload(
          {
            aria2cBin: bin,
            url: targetUrl,
            filename,
            outputDir: config.outDir,
            connections: config.connections,
          },
          {
            onProgress: progress =>
              setStage(s => (s.name === 'baking' ? {...s, progress} : s)),
          },
          ac.signal,
        )
        onOutcome({filepath})
        setStage({name: 'baked', filepath})
      } catch (err) {
        if (ac.signal.aborted) {
          setStage({name: 'input'})
        } else {
          setStage({name: 'error', message: (err as Error).message})
        }
      }
    },
    [cancel, config.outDir, config.connections, onOutcome],
  )

  const startTorrentBake = useCallback(
    async (target: string, name: string) => {
      cancel()
      setStage({name: 'baking', targetTitle: name, processing: false})

      const ac = new AbortController()
      abortRef.current = ac

      try {
        const bin = aria2cPathRef.current || (await findAria2c())
        if (!bin) throw new Error('aria2c is required for BitTorrent transfers.')

        const filepath = await bakeTorrentDownload(
          {
            aria2cBin: bin,
            target,
            outputDir: config.outDir,
          },
          {
            onProgress: progress =>
              setStage(s => (s.name === 'baking' ? {...s, progress} : s)),
          },
          ac.signal,
        )
        onOutcome({filepath})
        setStage({name: 'baked', filepath})
      } catch (err) {
        if (ac.signal.aborted) {
          setStage({name: 'input'})
        } else {
          setStage({name: 'error', message: (err as Error).message})
        }
      }
    },
    [cancel, config.outDir, onOutcome],
  )

  const probeSingle = useCallback(
    async (targetUrl: string, bin: string, signal: AbortSignal) => {
      setStage({name: 'probing', status: 'fetching video info…'})
      const result = await probeVideo(bin, targetUrl, signal)
      setMeta(result.meta)
      setCachedJsonPath(result.cachedJsonPath)
      const resolvedPortions = extractPortions(result.meta, {embedMetadata: config.embedMetadata})
      setPortions(resolvedPortions)
      setHistory(addToHistory(targetUrl))

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
    },
    [config, startBake],
  )

  const startProbe = useCallback(
    async (rawInput: string) => {
      cancel()
      setStage({name: 'probing', status: 'inspecting input…'})

      const ac = new AbortController()
      abortRef.current = ac

      try {
        const [aria2c, ffmpeg] = await Promise.all([findAria2c(), findFfmpeg()])
        aria2cPathRef.current = aria2c
        ffmpegRef.current = ffmpeg

        const inspection = await inspectTarget(rawInput, ac.signal)

        if (inspection.type === 'torrent') {
          setUrl(rawInput)
          setPlatform({key: 'bittorrent', label: 'BitTorrent'})
          setStage({name: 'torrent_prompt', title: inspection.name, target: inspection.target})
          setHistory(addToHistory(rawInput))
          return
        }

        if (inspection.type === 'direct') {
          setUrl(inspection.url)
          setPlatform({key: 'direct', label: 'Direct Download'})
          setStage({
            name: 'direct_prompt',
            filename: inspection.filename,
            size: inspection.size,
            url: inspection.url,
          })
          setHistory(addToHistory(inspection.url))
          return
        }

        // Video / Streaming
        const {cleanUrl, timeRange: tr, timeLabel: tl} = inspection
        setUrl(cleanUrl)
        setTimeRange(tr)
        setTimeLabel(tl)
        setPlatform(identifySite(cleanUrl))
        setIsPlaylistMode(false)
        setPlaylistMeta(null)
        setStage({name: 'probing', status: 'probing video…'})

        const bin = await ensureYtDlpBinary(
          msg => setStage(s => (s.name === 'probing' ? {...s, status: msg} : s)),
          ac.signal,
        )
        ytdlpRef.current = bin

        if (!tr && isPlaylistUrl(cleanUrl)) {
          setStage({name: 'probing', status: 'inspecting playlist…'})
          const pl = await probePlaylist(bin, cleanUrl, ac.signal)
          if (pl && pl.trackCount > 1) {
            setPlaylistMeta(pl)
            setStage({name: 'playlist_prompt', playlist: pl})
            return
          }
        }

        await probeSingle(cleanUrl, bin, ac.signal)
      } catch (err) {
        if (ac.signal.aborted) {
          setStage({name: 'input'})
        } else {
          setStage({name: 'error', message: (err as Error).message})
        }
      }
    },
    [cancel, probeSingle],
  )

  useEffect(() => {
    void findAria2c().then(path => {
      aria2cPathRef.current = path
    })
    void findFfmpeg().then(path => {
      ffmpegRef.current = path
    })
    if (initialUrl) void startProbe(initialUrl)
  }, [initialUrl, startProbe])

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
      return
    }

    if (key.escape) {
      if (stage.name === 'probing' || stage.name === 'baking') {
        cancel()
      } else if (
        stage.name === 'selecting' ||
        stage.name === 'input' ||
        stage.name === 'playlist_prompt' ||
        stage.name === 'direct_prompt' ||
        stage.name === 'torrent_prompt'
      ) {
        resetToInput()
      }
    }
    if (key.return && stage.name === 'error') {
      setStage({name: 'input'})
    }
    if (key.return && stage.name === 'baked') {
      resetToInput()
    }
  })

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
    const mascotRow = locateFrameRow(MASCOT_MATCH)
    if (mascotRow !== -1) {
      const bounds = frameRowBounds(mascotRow)
      if (bounds) {
        targets.push({
          match: MASCOT_MATCH,
          padY: 6,
          action: () => {
            cancel()
            setShowSettings(false)
            resetToInput()
          },
        })
      }
    }

    if (stage.name === 'input') {
      targets.push({
        match: BAKE_BUTTON,
        padY: 1,
        action: () => {
          if (url.trim() && isLikelyTarget(url.trim())) void startProbe(url.trim())
        },
      })
    }

    if (stage.name === 'baked') {
      targets.push({
        match: DONE_LABEL,
        action: resetToInput,
      })
    }

    targets.push({
      match: 'settings',
      action: () => setShowSettings(s => !s),
    })

    clickTargets.current = targets
  }, [stage, url, showSettings, cancel])

  const panelWidth = Math.min(64, cols - 4)

  return (
    <StageViewport>
      <AnpanMascot />
      <Gap />
      <Box flexDirection="column" alignItems="center">
        <Text>{TAGLINE}</Text>
        <Text dimColor={palette.dimAccent}>{SUPPORTED_HINT}</Text>
      </Box>
      <Gap />

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
          {stage.name === 'input' && (
            <>
              <TrayInput
                title="url / magnet / file"
                width={panelWidth}
                actionLabel={BAKE_BUTTON}
                actionDim={!url.trim()}
              >
                <KeyField
                  value={url}
                  onChange={handleUrlChange}
                  onSubmit={v => {
                    const trimmed = v.trim()
                    if (trimmed && isLikelyTarget(trimmed)) void startProbe(trimmed)
                  }}
                  placeholder={clipboardUrl ? `${clipboardUrl}  ⇥ paste` : 'https://... or magnet:?...'}
                  width={panelWidth - 8}
                  history={history}
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

          {stage.name === 'direct_prompt' && (
            <>
              <Box flexDirection="column" alignItems="center" width={panelWidth}>
                <Text bold>
                  {truncate(stage.filename, panelWidth)}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  {[stage.size ? formatBytes(stage.size) : '', `${config.connections} connections · direct file`]
                    .filter(Boolean)
                    .join(' · ')}
                </Text>
              </Box>
              <Gap />
              <BunCard title="direct file download" width={panelWidth}>
                <SelectInput
                  items={[
                    {label: `download file (${config.connections} connections)`, value: 'download'},
                    {label: 'cancel', value: 'cancel'},
                  ]}
                  onSelect={item => {
                    if (item.value === 'download') {
                      void startDirectBake(stage.url, stage.filename)
                    } else {
                      setStage({name: 'input'})
                    }
                  }}
                  indicatorComponent={PortionIndicator}
                  itemComponent={PortionItem}
                />
              </BunCard>
            </>
          )}

          {stage.name === 'torrent_prompt' && (
            <>
              <Box flexDirection="column" alignItems="center" width={panelWidth}>
                <Text bold>
                  {truncate(stage.title, panelWidth)}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  BitTorrent P2P transfer · aria2c
                </Text>
              </Box>
              <Gap />
              <BunCard title="bittorrent transfer" width={panelWidth}>
                <SelectInput
                  items={[
                    {label: 'start BitTorrent download (P2P)', value: 'download'},
                    {label: 'cancel', value: 'cancel'},
                  ]}
                  onSelect={item => {
                    if (item.value === 'download') {
                      void startTorrentBake(stage.target, stage.title)
                    } else {
                      setStage({name: 'input'})
                    }
                  }}
                  indicatorComponent={PortionIndicator}
                  itemComponent={PortionItem}
                />
              </BunCard>
            </>
          )}

          {stage.name === 'playlist_prompt' && (
            <>
              <Box flexDirection="column" alignItems="center" width={panelWidth}>
                <Text bold>
                  {truncate(stage.playlist.title, panelWidth)}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  {[stage.playlist.uploader, `${stage.playlist.trackCount} tracks`, platform?.label]
                    .filter(Boolean)
                    .join(' · ')}
                </Text>
              </Box>
              <Gap />
              <BunCard title="playlist detected" width={panelWidth}>
                <SelectInput
                  items={[
                    {label: `download full playlist (${stage.playlist.trackCount} tracks)`, value: 'full'},
                    {label: 'download single track only', value: 'single'},
                  ]}
                  onSelect={item => {
                    if (item.value === 'full') {
                      setIsPlaylistMode(true)
                      const plPortions = extractPlaylistPortions({embedMetadata: config.embedMetadata})
                      setPortions(plPortions)
                      setStage({name: 'selecting'})
                    } else {
                      setIsPlaylistMode(false)
                      const ac = new AbortController()
                      abortRef.current = ac
                      void probeSingle(url, ytdlpRef.current, ac.signal)
                    }
                  }}
                  indicatorComponent={PortionIndicator}
                  itemComponent={PortionItem}
                />
              </BunCard>
            </>
          )}

          {stage.name === 'selecting' && (
            <>
              <Box flexDirection="column" alignItems="center" width={panelWidth}>
                <Text bold>
                  {truncate(isPlaylistMode && playlistMeta ? playlistMeta.title : meta?.title ?? 'Media', panelWidth)}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  {(isPlaylistMode && playlistMeta
                    ? [playlistMeta.uploader, `${playlistMeta.trackCount} tracks`, platform?.label]
                    : [meta?.uploader, meta?.duration ? formatDuration(meta.duration) : '', platform?.label]
                  )
                    .filter(Boolean)
                    .join(' · ')}
                </Text>
                {timeLabel && (
                  <Text dimColor={palette.dimAccent}>
                    trim: <Text bold>{timeLabel}</Text>
                  </Text>
                )}
              </Box>
              <Gap />
              <BunCard title={isPlaylistMode ? 'playlist format' : 'format'} width={panelWidth}>
                <SelectInput
                  items={portions.map(p => ({label: p.label, value: p}))}
                  onSelect={item => void startBake(item.value)}
                  indicatorComponent={PortionIndicator}
                  itemComponent={PortionItem}
                />
              </BunCard>
            </>
          )}

          {stage.name === 'baking' && (
            <>
              <Text bold>
                {truncate(
                  stage.targetTitle ??
                    (isPlaylistMode && playlistMeta ? playlistMeta.title : meta?.title ?? 'Download'),
                  panelWidth,
                )}
              </Text>
              {timeLabel && (
                <Text dimColor={palette.dimAccent}>
                  trim: <Text bold>{timeLabel}</Text>
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
                        {stage.progress.seeders !== undefined
                          ? `  · P2P (${stage.progress.connections} peers, ${stage.progress.seeders} seeds)`
                          : config.aria2c && aria2cPathRef.current
                            ? `  · aria2c (${config.connections})`
                            : ''}
                      </Text>
                    </>
                  ) : (
                    <Text dimColor={palette.dimAccent}>
                      {indeterminateMeta(stage.progress)}
                      {stage.progress.seeders !== undefined
                        ? `  · P2P (${stage.progress.connections} peers, ${stage.progress.seeders} seeds)`
                        : config.aria2c && aria2cPathRef.current
                          ? `  · aria2c (${config.connections})`
                          : ''}
                    </Text>
                  )}
                </>
              ) : (
                <Box>
                  <Text>
                    <Spinner type="dots" />
                  </Text>
                  <Text dimColor={palette.dimAccent}>
                    {' downloading…'}
                  </Text>
                </Box>
              )}
            </>
          )}

          {stage.name === 'baked' && (
            <>
              <Text bold>
                {isPlaylistMode ? 'playlist downloaded' : 'downloaded'}
              </Text>
              <Text dimColor={palette.dimAccent}>
                {shortenPath(stage.filepath, os.homedir(), panelWidth)}
              </Text>
              <Gap />
              <Text>{DONE_LABEL}</Text>
            </>
          )}

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
