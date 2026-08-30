import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

const CACHE_FILE = path.join(os.homedir(), '.config', 'anpan', 'update-cache.json')
const CACHE_TTL_MS = 4 * 60 * 60 * 1000 // 4 hours
const REGISTRY_URL = 'https://registry.npmjs.org/anpan-cli/latest'

export type UpdateCheckResult = {
  updateAvailable: boolean
  latestVersion: string
  currentVersion: string
}

type UpdateCache = {
  lastChecked: number
  latestVersion: string
}

export function isNewerVersion(remote: string, current: string): boolean {
  const parse = (v: string) =>
    v
      .replace(/^v/, '')
      .split('.')
      .map(part => parseInt(part, 10) || 0)

  const [rMaj = 0, rMin = 0, rPatch = 0] = parse(remote)
  const [cMaj = 0, cMin = 0, cPatch = 0] = parse(current)

  if (rMaj !== cMaj) return rMaj > cMaj
  if (rMin !== cMin) return rMin > cMin
  return rPatch > cPatch
}

function readCache(): UpdateCache | null {
  try {
    const data = JSON.parse(fs.readFileSync(CACHE_FILE, 'utf8')) as UpdateCache
    if (typeof data.lastChecked === 'number' && typeof data.latestVersion === 'string') {
      return data
    }
  } catch {}
  return null
}

function writeCache(latestVersion: string): void {
  try {
    fs.mkdirSync(path.dirname(CACHE_FILE), {recursive: true})
    fs.writeFileSync(
      CACHE_FILE,
      JSON.stringify({lastChecked: Date.now(), latestVersion}, null, 2),
      'utf8',
    )
  } catch {}
}

export async function checkUpdate(
  currentVersion: string,
  opts?: {force?: boolean; timeoutMs?: number},
): Promise<UpdateCheckResult | null> {
  const timeoutMs = opts?.timeoutMs ?? 1500
  const now = Date.now()

  if (!opts?.force) {
    const cached = readCache()
    if (cached && now - cached.lastChecked < CACHE_TTL_MS) {
      return {
        updateAvailable: isNewerVersion(cached.latestVersion, currentVersion),
        latestVersion: cached.latestVersion,
        currentVersion,
      }
    }
  }

  try {
    const res = await fetch(REGISTRY_URL, {
      signal: AbortSignal.timeout(timeoutMs),
      headers: {Accept: 'application/json'},
    })
    if (!res.ok) return null
    const data = (await res.json()) as {version?: string}
    if (!data.version) return null

    writeCache(data.version)
    return {
      updateAvailable: isNewerVersion(data.version, currentVersion),
      latestVersion: data.version,
      currentVersion,
    }
  } catch {
    // Offline or network timeout: fall back to cached value if present
    const cached = readCache()
    if (cached) {
      return {
        updateAvailable: isNewerVersion(cached.latestVersion, currentVersion),
        latestVersion: cached.latestVersion,
        currentVersion,
      }
    }
    return null
  }
}
