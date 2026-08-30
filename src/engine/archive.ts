export type ArchiveFile = {
  name: string
  url: string
}

export type ArchivePost = {
  title: string
  service: string
  user: string
  id: string
  files: ArchiveFile[]
  webpage_url: string
}

export const ARCHIVE_POST_REGEX =
  /^(?:https?:\/\/)?(?:www\.)?(kemono\.(?:cr|su|party)|coomer\.(?:su|party|st)|pawchive\.(?:st|pw))\/(?:api\/v1\/)?([^/?#]+)\/user\/([^/?#]+)\/post\/([^/?#]+)/i

export function isArchivePostUrl(url: string): boolean {
  return ARCHIVE_POST_REGEX.test(url.trim())
}

export function parseArchiveUrl(url: string): {
  domain: string
  service: string
  user: string
  id: string
} | null {
  const match = ARCHIVE_POST_REGEX.exec(url.trim())
  if (!match) return null
  return {
    domain: match[1]!.toLowerCase(),
    service: match[2]!,
    user: match[3]!,
    id: match[4]!,
  }
}

export async function probeArchivePost(
  url: string,
  signal?: AbortSignal,
): Promise<ArchivePost | null> {
  const parsed = parseArchiveUrl(url)
  if (!parsed) return null

  const {domain, service, user, id} = parsed
  const isPawchive = domain.startsWith('pawchive')
  const isCoomer = domain.startsWith('coomer')

  const apiHost = isPawchive
    ? 'https://pawchive.pw'
    : isCoomer
      ? 'https://coomer.st'
      : 'https://kemono.cr'

  const endpoint = `${apiHost}/api/v1/${service}/user/${user}/post/${id}`

  try {
    const timeoutSignal = signal
      ? AbortSignal.any([signal, AbortSignal.timeout(10_000)])
      : AbortSignal.timeout(10_000)

    const resp = await fetch(endpoint, {
      signal: timeoutSignal,
      headers: {
        Accept: isPawchive ? 'application/json' : 'text/css,application/json',
        'Accept-Encoding': 'identity',
      },
    })

    if (!resp.ok) return null

    const json = (await resp.json()) as {
      post?: Record<string, unknown>
      [key: string]: unknown
    }

    const post = (json.post ?? json) as {
      title?: string
      user?: string
      service?: string
      id?: string
      file?: {name?: string; path?: string}
      attachments?: Array<{name?: string; path?: string}>
    }

    const files: ArchiveFile[] = []
    const seenPaths = new Set<string>()

    const addFile = (rawFile?: {name?: string; path?: string}) => {
      if (!rawFile?.path || seenPaths.has(rawFile.path)) return
      seenPaths.add(rawFile.path)

      const fallbackName = rawFile.path.split('/').pop() ?? 'file'
      const cleanName = (rawFile.name || fallbackName).replace(/[/\\?%*:|"<>]/g, '_')

      let fileUrl = ''
      if (isPawchive) {
        fileUrl = `https://file.pawchive.pw/data${rawFile.path}?f=${encodeURIComponent(cleanName)}`
      } else {
        const cdnDomain = isCoomer ? 'https://coomer.st' : 'https://kemono.cr'
        fileUrl = `${cdnDomain}/data${rawFile.path}?f=${encodeURIComponent(cleanName)}`
      }

      files.push({name: cleanName, url: fileUrl})
    }

    if (post.file) addFile(post.file)
    if (Array.isArray(post.attachments)) {
      for (const att of post.attachments) addFile(att)
    }

    const rawTitle = (post.title || `${service} post by ${user}`).trim()
    const cleanTitle = rawTitle.replace(/[/\\?%*:|"<>]/g, '_') || 'Archive Post'

    return {
      title: cleanTitle,
      service: post.service || service,
      user: post.user || user,
      id: post.id || id,
      files,
      webpage_url: url,
    }
  } catch {
    return null
  }
}
