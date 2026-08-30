import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

const HISTORY_PATH = path.join(os.homedir(), '.config', 'anpan', 'history.json')
const MAX_ENTRIES = 50

export function loadHistory(): string[] {
  try {
    const data: unknown = JSON.parse(fs.readFileSync(HISTORY_PATH, 'utf8'))
    return Array.isArray(data) ? data.filter((e): e is string => typeof e === 'string') : []
  } catch {
    return []
  }
}

export function addToHistory(url: string): string[] {
  const updated = [url, ...loadHistory().filter(e => e !== url)].slice(0, MAX_ENTRIES)
  try {
    fs.mkdirSync(path.dirname(HISTORY_PATH), {recursive: true})
    fs.writeFileSync(HISTORY_PATH, `${JSON.stringify(updated, null, 2)}\n`)
  } catch {}
  return updated
}
