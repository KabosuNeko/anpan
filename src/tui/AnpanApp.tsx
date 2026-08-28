import React, {useCallback, useEffect, useRef, useState} from 'react'
import {Box, Text, useApp, useInput, useStdout} from 'ink'
import {AnpanMascot} from './mascot/AnpanMascot.js'
import {InfoPane} from './panes/InfoPane.js'
import {ActionPane} from './panes/ActionPane.js'
import {FooterKeys} from './primitives/FooterKeys.js'
import {StageViewport} from './primitives/StageViewport.js'
import {tapTargetAt, locateFrameRow, frameRowBounds, type TapTarget} from './events/hitTest.js'
import {usePointer} from './events/usePointer.js'
import {ThemeProvider, useAnpanTheme} from './theme/palette.js'
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

const TAGLINE = 'minimal terminal video downloader'

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

function AppContent({initialUrl, clipboardUrl, onOutcome}: AnpanAppProps) {
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

  const reset = useCallback(() => {
    setUrl('')
    setMeta(null)
    setPortions([])
    setPlatform(null)
    setCachedJsonPath(undefined)
    setStage({name: 'input'})
  }, [])

  const handleUrlChange = useCallback((newUrl: string) => {
    setUrl(newUrl)
    if (!newUrl.trim()) {
      setMeta(null)
      setPlatform(null)
      setPortions([])
      setCachedJsonPath(undefined)
      if (stage.name === 'selecting' || stage.name === 'error') {
        setStage({name: 'input'})
      }
    }
  }, [stage.name])

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
      return
    }

    if (key.escape) {
      if (stage.name === 'probing' || stage.name === 'baking') {
        cancel()
      } else if (stage.name === 'selecting' || stage.name === 'input') {
        reset()
      }
    }
    if (key.return && stage.name === 'error') {
      setStage({name: 'input'})
    }
    if (key.return && stage.name === 'baked') {
      reset()
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
            reset()
          },
        })
      }
    }

    if (stage.name === 'input') {
      targets.push({
        match: 'bake',
        padY: 1,
        action: () => {
          if (url.trim() && isLikelyUrl(url.trim())) void startProbe(url.trim())
        },
      })
    }

    if (stage.name === 'baked') {
      targets.push({
        match: 'download another',
        action: reset,
      })
    }

    targets.push({
      match: 'settings',
      action: () => setShowSettings(s => !s),
    })

    clickTargets.current = targets
  }, [stage, url, showSettings, reset, cancel, startProbe])

  // responsive layout dimensions
  const totalWidth = Math.min(78, Math.max(68, cols - 2))
  const leftWidth = 28
  const rightWidth = totalWidth - leftWidth - 1
  const dashboardHeight = 14

  return (
    <StageViewport>
      <AnpanMascot />
      <Box height={1}>
        <Text dimColor={palette.dimAccent}>{TAGLINE}</Text>
      </Box>

      {/* ── 2-Column Split Dashboard ────────────────────────── */}
      <Box flexDirection="row" width={totalWidth} gap={1} alignItems="stretch">
        <InfoPane
          width={leftWidth}
          height={dashboardHeight}
          stageName={stage.name}
          meta={meta}
          platform={platform}
          config={config}
          hasAria2c={Boolean(aria2cPathRef.current)}
        />

        <ActionPane
          width={rightWidth}
          height={dashboardHeight}
          stage={stage}
          url={url}
          onUrlChange={handleUrlChange}
          onUrlSubmit={v => {
            const trimmed = v.trim()
            if (trimmed && isLikelyUrl(trimmed)) void startProbe(trimmed)
          }}
          clipboardUrl={clipboardUrl}
          history={history}
          portions={portions}
          onPortionSelect={startBake}
          showSettings={showSettings}
          config={config}
          onConfigChange={updated => {
            setConfig(updated)
            saveConfig(updated)
          }}
          onCloseSettings={() => setShowSettings(false)}
          onReset={reset}
        />
      </Box>

      {/* ── Footer ─────────────────────────────────────────── */}
      <Box height={1}>
        {showSettings ? (
          <FooterKeys hints={[['↑↓', 'select'], ['↵', 'edit/toggle'], ['⇄', 'preset'], ['esc', 'close']]} />
        ) : (
          <FooterKeys hints={HINTS[stage.name]} />
        )}
      </Box>
    </StageViewport>
  )
}
