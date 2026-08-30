import {useCallback, useEffect, useRef, useState} from 'react'
import os from 'node:os'
import path from 'node:path'
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
import {tapTargetAt, locateFrameRow, type TapTarget} from './events/hitTest.js'
import {usePointer} from './events/usePointer.js'
import {ThemeProvider, useAnpanTheme} from './theme/palette.js'
import {
  formatBytes,
  formatDuration,
  formatEta,
  formatSpeed,
  resolveUserPath,
  shortenPath,
  truncate,
  wrapText,
} from '../core/units.js'
import {addToHistory, loadHistory} from '../system/history.js'
import {loadConfig, saveConfig} from '../system/config.js'
import {checkUpdate} from '../system/update.js'
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
import {findAria2c, buildAria2cArgs, bakeDirectDownload, bakeTorrentDownload, bakeBatchDownload} from '../engine/aria2c.js'
import type {ArchivePost} from '../engine/archive.js'

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
    tag = `[${progress.playlistItem}/${progress.playlistTotal}] `
  } else if (progress.totalParts > 1) {
    tag = `[${progress.part}/${progress.totalParts}] `
  }
  return tag
}

function progressMeta(progress: BakeProgress): string {
  const speed = progress.speed ? formatSpeed(progress.speed) : ''
  const eta = progress.eta ? `${formatEta(progress.eta)} left` : ''
  return `${partTag(progress)}${speed.padStart(10)}  ${eta.padEnd(12)}`
}

function indeterminateMeta(progress: BakeProgress): string {
  const bytes = progress.downloadedBytes > 0 ? formatBytes(progress.downloadedBytes) : 'connecting…'
  const speed = progress.speed ? formatSpeed(progress.speed) : ''
  return `${partTag(progress)}${bytes.padStart(8)}${speed ? `  ${speed.padEnd(10)}` : ''}`
}

export type Outcome = {filepath?: string}

type Stage =
  | {name: 'input'}
  | {name: 'probing'; status: string}
  | {name: 'playlist_prompt'; playlist: PlaylistMeta}
  | {name: 'direct_prompt'; filename: string; size?: number; url: string}
  | {name: 'torrent_prompt'; title: string; target: string}
  | {name: 'selecting'}
  | {
      name: 'dest_prompt'
      portion?: Portion
      directTarget?: {url: string; filename: string}
      torrentTarget?: {target: string; title: string}
      archiveTarget?: {title: string; items: Array<{name: string; url: string; mirrors?: string[]}>}
    }
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
  dest_prompt: [
    ['↑↓', 'choose'],
    ['↵', 'confirm'],
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
  initialOutDir?: string
  clipboardUrl?: string
  version?: string
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
  initialOutDir,
  clipboardUrl,
  version,
  onOutcome,
}: AnpanAppProps) {
  const palette = useAnpanTheme()
  const {exit} = useApp()
  const {stdout} = useStdout()
  const cols = stdout?.columns ?? 80

  const [latestVersion, setLatestVersion] = useState<string | null>(null)

  useEffect(() => {
    if (!version) return
    let active = true
    checkUpdate(version)
      .then(res => {
        if (active && res?.updateAvailable) {
          setLatestVersion(res.latestVersion)
        }
      })
      .catch(() => {})
    return () => {
      active = false
    }
  }, [version])

  const [config, setConfig] = useState(loadConfig)
  const [showSettings, setShowSettings] = useState(false)

  const [url, setUrl] = useState(initialUrl ?? '')
  const [timeRange, setTimeRange] = useState<string | undefined>()
  const [timeLabel, setTimeLabel] = useState<string | undefined>()
  const [isPlaylistMode, setIsPlaylistMode] = useState(false)
  const [playlistMeta, setPlaylistMeta] = useState<PlaylistMeta | null>(null)
  const [archivePost, setArchivePost] = useState<ArchivePost | null>(null)
  const [isCustomDirInput, setIsCustomDirInput] = useState(false)
  const [customDirInput, setCustomDirInput] = useState('')

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
    setArchivePost(null)
    setPlatform(null)
    setCachedJsonPath(undefined)
    setTimeRange(undefined)
    setTimeLabel(undefined)
    setIsPlaylistMode(false)
    setPlaylistMeta(null)
    setIsCustomDirInput(false)
    setStage({name: 'input'})
  }, [])

  const handleUrlChange = useCallback((newUrl: string) => {
    setUrl(newUrl)
    if (!newUrl.trim()) {
      resetToInput()
    }
  }, [resetToInput])

  const startBake = useCallback(
    async (portion: Portion, targetOutputDir?: string) => {
      cancel()
      const effectiveOutDir = resolveUserPath(targetOutputDir || initialOutDir || config.outDir)
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
            outputDir: effectiveOutDir,
            timeRange,
            isPlaylist: isPlaylistMode,
            cookiesBrowser: config.cookiesBrowser,
            subtitles: config.subtitles,
            subLangs: config.subLangs,
            sponsorBlock: config.sponsorBlock,
            writeThumbnail: config.writeThumbnail,
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
    [cancel, initialOutDir, config, url, cachedJsonPath, isPlaylistMode, timeRange, onOutcome],
  )

  const startDirectBake = useCallback(
    async (targetUrl: string, filename: string, targetOutputDir?: string) => {
      cancel()
      const effectiveOutDir = resolveUserPath(targetOutputDir || initialOutDir || config.outDir)
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
            outputDir: effectiveOutDir,
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
    [cancel, initialOutDir, config.outDir, config.connections, onOutcome],
  )

  const startTorrentBake = useCallback(
    async (target: string, name: string, targetOutputDir?: string) => {
      cancel()
      const effectiveOutDir = resolveUserPath(targetOutputDir || initialOutDir || config.outDir)
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
            outputDir: effectiveOutDir,
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
    [cancel, initialOutDir, config.outDir, onOutcome],
  )

  const startArchiveBake = useCallback(
    async (
      title: string,
      items: Array<{name: string; url: string; mirrors?: string[]}>,
      targetOutputDir?: string,
    ) => {
      cancel()
      const ac = new AbortController()
      abortRef.current = ac

      setStage({
        name: 'baking',
        targetTitle: title,
        processing: false,
      })

      try {
        const baseOutDir = resolveUserPath(targetOutputDir || initialOutDir || config.outDir)
        const effectiveOutDir =
          items.length > 1 ? path.join(baseOutDir, title) : baseOutDir

        const bin = aria2cPathRef.current || (await findAria2c())
        if (!bin) throw new Error('aria2c is required for downloading archive posts.')

        const filepath = await bakeBatchDownload(
          {
            aria2cBin: bin,
            items: items.map(it => ({url: it.url, mirrors: it.mirrors, filename: it.name})),
            outputDir: effectiveOutDir,
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
    [cancel, initialOutDir, config.outDir, config.connections, onOutcome],
  )

  const triggerArchiveDownload = useCallback(
    (title: string, items: Array<{name: string; url: string; mirrors?: string[]}>) => {
      if (config.askSaveDir && !initialOutDir) {
        setIsCustomDirInput(false)
        setCustomDirInput(shortenPath(config.outDir, os.homedir()))
        setStage({name: 'dest_prompt', archiveTarget: {title, items}})
      } else {
        void startArchiveBake(title, items)
      }
    },
    [config.askSaveDir, config.outDir, initialOutDir, startArchiveBake],
  )

  const triggerDirectDownload = useCallback(
    (targetUrl: string, filename: string) => {
      if (config.askSaveDir && !initialOutDir) {
        setIsCustomDirInput(false)
        setCustomDirInput(shortenPath(config.outDir, os.homedir()))
        setStage({
          name: 'dest_prompt',
          directTarget: {url: targetUrl, filename},
        })
      } else {
        void startDirectBake(targetUrl, filename)
      }
    },
    [config.askSaveDir, config.outDir, initialOutDir, startDirectBake],
  )

  const triggerPortionDownload = useCallback(
    (portion: Portion) => {
      if (config.askSaveDir && !initialOutDir) {
        setIsCustomDirInput(false)
        setCustomDirInput(shortenPath(config.outDir, os.homedir()))
        setStage({name: 'dest_prompt', portion})
      } else {
        void startBake(portion)
      }
    },
    [config.askSaveDir, config.outDir, initialOutDir, startBake],
  )

  const confirmDestAndDownload = useCallback(
    (dir: string) => {
      const resolved = resolveUserPath(dir)

      if (stage.name === 'dest_prompt') {
        if (stage.portion) {
          void startBake(stage.portion, resolved)
        } else if (stage.directTarget) {
          void startDirectBake(stage.directTarget.url, stage.directTarget.filename, resolved)
        } else if (stage.torrentTarget) {
          void startTorrentBake(stage.torrentTarget.target, stage.torrentTarget.title, resolved)
        } else if (stage.archiveTarget) {
          void startArchiveBake(stage.archiveTarget.title, stage.archiveTarget.items, resolved)
        }
      }
    },
    [stage, startBake, startDirectBake, startTorrentBake, startArchiveBake],
  )

  const probeSingle = useCallback(
    async (targetUrl: string, bin: string, signal: AbortSignal) => {
      setStage({name: 'probing', status: 'fetching media info…'})
      const result = await probeVideo(bin, targetUrl, signal)
      setMeta(result.meta)
      setCachedJsonPath(result.cachedJsonPath)
      const resolvedPortions = extractPortions(result.meta, {
        embedMetadata: config.embedMetadata,
        videoContainer: config.videoContainer,
        audioFormat: config.audioFormat,
      })
      setPortions(resolvedPortions)
      setHistory(addToHistory(targetUrl))

      if (config.preferQuality === 'best' && resolvedPortions[0]) {
        triggerPortionDownload(resolvedPortions[0])
      } else if (config.preferQuality === 'audio') {
        const audioChoice = resolvedPortions.find(p => p.kind === 'audio')
        if (audioChoice) triggerPortionDownload(audioChoice)
        else setStage({name: 'selecting'})
      } else if (config.preferQuality === '1080p') {
        const p1080 = resolvedPortions.find(p => p.label.startsWith('1080p'))
        if (p1080) triggerPortionDownload(p1080)
        else setStage({name: 'selecting'})
      } else {
        setStage({name: 'selecting'})
      }
    },
    [config, triggerPortionDownload],
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

        if (inspection.type === 'archive') {
          const {post} = inspection
          setUrl(post.webpage_url)
          setPlatform(identifySite(post.webpage_url))
          setHistory(addToHistory(post.webpage_url))

          if (post.files.length === 1) {
            setArchivePost(post)
            triggerArchiveDownload(post.title, post.files)
            return
          }

          setArchivePost(post)
          const archivePortions: Portion[] = [
            {
              kind: 'video',
              label: `📦 all files (${post.files.length} items) · ${post.title}`,
              ytdlpArgs: [],
            },
            ...post.files.map(f => ({
              kind: 'video' as const,
              label: `📄 ${f.name}`,
              ytdlpArgs: [],
            })),
          ]
          setPortions(archivePortions)
          setStage({name: 'selecting'})
          return
        }

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
      } else if (stage.name === 'dest_prompt') {
        if (isCustomDirInput) {
          setIsCustomDirInput(false)
        } else if (portions.length > 0) {
          setStage({name: 'selecting'})
        } else {
          resetToInput()
        }
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
    if (locateFrameRow(MASCOT_MATCH) !== -1) {
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
        {latestVersion && (stage.name === 'input' || stage.name === 'baked') && (
          <Box marginTop={1}>
            <Text color="yellow">✦ update available: </Text>
            <Text dimColor>{version}</Text>
            <Text color="yellow"> → </Text>
            <Text bold color="green">{latestVersion}</Text>
            <Text dimColor> (run: npm i -g anpan-cli)</Text>
          </Box>
        )}
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
                      triggerDirectDownload(stage.url, stage.filename)
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
                      if (config.askSaveDir && !initialOutDir) {
                        setIsCustomDirInput(false)
                        setCustomDirInput(shortenPath(config.outDir, os.homedir()))
                        setStage({
                          name: 'dest_prompt',
                          torrentTarget: {target: stage.target, title: stage.title},
                        })
                      } else {
                        void startTorrentBake(stage.target, stage.title)
                      }
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
                      const plPortions = extractPlaylistPortions({
                        embedMetadata: config.embedMetadata,
                        videoContainer: config.videoContainer,
                        audioFormat: config.audioFormat,
                      })
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
                  {truncate(
                    archivePost
                      ? archivePost.title
                      : isPlaylistMode && playlistMeta
                        ? playlistMeta.title
                        : meta?.title ?? 'Media',
                    panelWidth,
                  )}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  {(archivePost
                    ? [archivePost.service, archivePost.user, `${archivePost.files.length} files`, platform?.label]
                    : isPlaylistMode && playlistMeta
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
              <BunCard
                title={archivePost ? 'files to download' : isPlaylistMode ? 'playlist format' : 'format'}
                width={panelWidth}
              >
                <SelectInput
                  items={portions.map(p => ({label: p.label, value: p}))}
                  onSelect={item => {
                    if (archivePost) {
                      const idx = portions.findIndex(p => p === item.value)
                      if (idx <= 0) {
                        triggerArchiveDownload(archivePost.title, archivePost.files)
                      } else {
                        const file = archivePost.files[idx - 1]
                        if (file) triggerArchiveDownload(archivePost.title, [file])
                      }
                    } else {
                      triggerPortionDownload(item.value)
                    }
                  }}
                  indicatorComponent={PortionIndicator}
                  itemComponent={PortionItem}
                />
              </BunCard>
            </>
          )}

          {stage.name === 'dest_prompt' && (
            <>
              <Box flexDirection="column" alignItems="center" width={panelWidth}>
                <Text bold>
                  {truncate(
                    stage.portion
                      ? (archivePost
                          ? archivePost.title
                          : isPlaylistMode && playlistMeta
                            ? playlistMeta.title
                            : meta?.title ?? 'Download')
                      : stage.archiveTarget?.title ?? stage.directTarget?.filename ?? stage.torrentTarget?.title ?? 'Download',
                    panelWidth,
                  )}
                </Text>
                <Text dimColor={palette.dimAccent}>
                  {stage.portion
                    ? stage.portion.label
                    : stage.archiveTarget
                      ? `${stage.archiveTarget.items.length} files (will be saved into subfolder)`
                      : 'select destination folder'}
                </Text>
              </Box>
              <Gap />
              <BunCard title="choose save folder" width={panelWidth}>
                {isCustomDirInput ? (
                  <Box flexDirection="column" gap={0} width="100%">
                    <Text dimColor={palette.dimAccent}>enter directory path (↵ confirm, esc back):</Text>
                    <Box width={panelWidth - 8} height={1}>
                      <KeyField
                        value={customDirInput}
                        onChange={setCustomDirInput}
                        onSubmit={confirmDestAndDownload}
                        width={panelWidth - 10}
                      />
                    </Box>
                  </Box>
                ) : (
                  <SelectInput
                    items={[
                      {label: `[↵] ${shortenPath(config.outDir, os.homedir(), 28)} (default)`, value: config.outDir},
                      ...(path.resolve(config.outDir) !== path.resolve(path.join(os.homedir(), 'Downloads'))
                        ? [{label: `[D] ${shortenPath(path.join(os.homedir(), 'Downloads'), os.homedir(), 28)}`, value: path.join(os.homedir(), 'Downloads')}]
                        : []),
                      ...(path.resolve(config.outDir) !== path.resolve(path.join(os.homedir(), 'Videos'))
                        ? [{label: `[V] ${shortenPath(path.join(os.homedir(), 'Videos'), os.homedir(), 28)}`, value: path.join(os.homedir(), 'Videos')}]
                        : []),
                      ...(path.resolve(config.outDir) !== path.resolve(process.cwd())
                        ? [{label: `[C] Current folder (./)`, value: process.cwd()}]
                        : []),
                      {label: '[O] Custom path…', value: '__custom__'},
                    ]}
                    onSelect={item => {
                      if (item.value === '__custom__') {
                        setIsCustomDirInput(true)
                      } else {
                        confirmDestAndDownload(item.value)
                      }
                    }}
                    indicatorComponent={PortionIndicator}
                    itemComponent={PortionItem}
                  />
                )}
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
                    <Box alignItems="center">
                      {stage.progress.downloadedBytes === 0 && (
                        <Box marginRight={1}>
                          <Spinner type="dots" />
                        </Box>
                      )}
                      <Text dimColor={palette.dimAccent}>
                        {indeterminateMeta(stage.progress)}
                        {stage.progress.seeders !== undefined
                          ? `  · P2P (${stage.progress.connections} peers, ${stage.progress.seeders} seeds)`
                          : config.aria2c && aria2cPathRef.current
                            ? `  · aria2c (${config.connections})`
                            : ''}
                      </Text>
                    </Box>
                  )}
                </>
              ) : (
                <Box alignItems="center">
                  <Box marginRight={1}>
                    <Spinner type="dots" />
                  </Box>
                  <Text dimColor={palette.dimAccent}>
                    connecting to server…
                    {config.aria2c && aria2cPathRef.current ? `  · aria2c (${config.connections})` : ''}
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
