import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

export type AnpanConfig = {
  aria2c: boolean
  connections: 4 | 8 | 16 | 32
  askSaveDir: boolean
  outDir: string
  preferQuality: 'ask' | 'best' | '1080p' | 'audio'
  videoContainer: 'mp4' | 'mkv' | 'webm'
  audioFormat: 'mp3' | 'm4a' | 'opus' | 'flac' | 'wav'
  cookiesBrowser: 'none' | 'chrome' | 'firefox' | 'brave' | 'edge' | 'safari'
  subtitles: 'off' | 'embed' | 'write'
  subLangs: 'vi,en' | 'all' | 'en'
  sponsorBlock: 'off' | 'remove' | 'mark'
  embedMetadata: boolean
  writeThumbnail: boolean
}

const CONFIG_PATH = path.join(os.homedir(), '.config', 'anpan', 'config.json')

const DEFAULT_CONFIG: AnpanConfig = {
  aria2c: true,
  connections: 16,
  askSaveDir: true,
  outDir: path.join(os.homedir(), 'Downloads'),
  preferQuality: 'ask',
  videoContainer: 'mp4',
  audioFormat: 'mp3',
  cookiesBrowser: 'none',
  subtitles: 'off',
  subLangs: 'vi,en',
  sponsorBlock: 'off',
  embedMetadata: true,
  writeThumbnail: false,
}

export function loadConfig(): AnpanConfig {
  try {
    const raw = fs.readFileSync(CONFIG_PATH, 'utf8')
    const parsed = JSON.parse(raw) as Partial<AnpanConfig>
    return {
      aria2c: typeof parsed.aria2c === 'boolean' ? parsed.aria2c : DEFAULT_CONFIG.aria2c,
      connections: [4, 8, 16, 32].includes(parsed.connections as number)
        ? (parsed.connections as 4 | 8 | 16 | 32)
        : DEFAULT_CONFIG.connections,
      askSaveDir:
        typeof parsed.askSaveDir === 'boolean' ? parsed.askSaveDir : DEFAULT_CONFIG.askSaveDir,
      embedMetadata:
        typeof parsed.embedMetadata === 'boolean' ? parsed.embedMetadata : DEFAULT_CONFIG.embedMetadata,
      writeThumbnail:
        typeof parsed.writeThumbnail === 'boolean' ? parsed.writeThumbnail : DEFAULT_CONFIG.writeThumbnail,
      outDir: typeof parsed.outDir === 'string' && parsed.outDir.trim() ? parsed.outDir : DEFAULT_CONFIG.outDir,
      preferQuality: ['ask', 'best', '1080p', 'audio'].includes(parsed.preferQuality as string)
        ? (parsed.preferQuality as AnpanConfig['preferQuality'])
        : DEFAULT_CONFIG.preferQuality,
      videoContainer: ['mp4', 'mkv', 'webm'].includes(parsed.videoContainer as string)
        ? (parsed.videoContainer as AnpanConfig['videoContainer'])
        : DEFAULT_CONFIG.videoContainer,
      audioFormat: ['mp3', 'm4a', 'opus', 'flac', 'wav'].includes(parsed.audioFormat as string)
        ? (parsed.audioFormat as AnpanConfig['audioFormat'])
        : DEFAULT_CONFIG.audioFormat,
      cookiesBrowser: ['none', 'chrome', 'firefox', 'brave', 'edge', 'safari'].includes(
        parsed.cookiesBrowser as string,
      )
        ? (parsed.cookiesBrowser as AnpanConfig['cookiesBrowser'])
        : DEFAULT_CONFIG.cookiesBrowser,
      subtitles: ['off', 'embed', 'write'].includes(parsed.subtitles as string)
        ? (parsed.subtitles as AnpanConfig['subtitles'])
        : DEFAULT_CONFIG.subtitles,
      subLangs: ['vi,en', 'all', 'en'].includes(parsed.subLangs as string)
        ? (parsed.subLangs as AnpanConfig['subLangs'])
        : DEFAULT_CONFIG.subLangs,
      sponsorBlock: ['off', 'remove', 'mark'].includes(parsed.sponsorBlock as string)
        ? (parsed.sponsorBlock as AnpanConfig['sponsorBlock'])
        : DEFAULT_CONFIG.sponsorBlock,
    }
  } catch {
    return {...DEFAULT_CONFIG}
  }
}

export function saveConfig(cfg: AnpanConfig): void {
  try {
    fs.mkdirSync(path.dirname(CONFIG_PATH), {recursive: true})
    fs.writeFileSync(CONFIG_PATH, `${JSON.stringify(cfg, null, 2)}\n`)
  } catch {}
}
