import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

export type AnpanConfig = {
  aria2c: boolean
  connections: 4 | 8 | 16 | 32
  embedMetadata: boolean
  outDir: string
  preferQuality: 'ask' | 'best' | '1080p' | 'audio'
}

const CONFIG_PATH = path.join(os.homedir(), '.config', 'anpan', 'config.json')

export const DEFAULT_CONFIG: AnpanConfig = {
  aria2c: true,
  connections: 16,
  embedMetadata: true,
  outDir: path.join(os.homedir(), 'Downloads'),
  preferQuality: 'ask',
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
      embedMetadata:
        typeof parsed.embedMetadata === 'boolean' ? parsed.embedMetadata : DEFAULT_CONFIG.embedMetadata,
      outDir: typeof parsed.outDir === 'string' && parsed.outDir.trim() ? parsed.outDir : DEFAULT_CONFIG.outDir,
      preferQuality: ['ask', 'best', '1080p', 'audio'].includes(parsed.preferQuality as string)
        ? (parsed.preferQuality as AnpanConfig['preferQuality'])
        : DEFAULT_CONFIG.preferQuality,
    }
  } catch {
    return {...DEFAULT_CONFIG}
  }
}

export function saveConfig(cfg: AnpanConfig): void {
  try {
    fs.mkdirSync(path.dirname(CONFIG_PATH), {recursive: true})
    fs.writeFileSync(CONFIG_PATH, `${JSON.stringify(cfg, null, 2)}\n`)
  } catch {
    // optional persistence
  }
}
