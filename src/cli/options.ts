type ParsedArgs =
  | {help: true; version: false; error?: undefined; initialUrl?: undefined; initialOutDir?: undefined}
  | {help: false; version: true; error?: undefined; initialUrl?: undefined; initialOutDir?: undefined}
  | {help: false; version: false; error: string; initialUrl?: undefined; initialOutDir?: undefined}
  | {help: false; version: false; error?: undefined; initialUrl?: string; initialOutDir?: string}

export function parseArgs(argv: string[]): ParsedArgs {
  let initialUrl: string | undefined
  let initialOutDir: string | undefined
  const positionals: string[] = []

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]!
    if (arg === '-h' || arg === '--help') return {help: true, version: false}
    if (arg === '-v' || arg === '--version') return {help: false, version: true}

    if (arg === '-o' || arg === '--output' || arg === '--out-dir') {
      const next = argv[++i]
      if (!next || next.startsWith('-')) {
        return {help: false, version: false, error: `option "${arg}" requires an output directory argument.`}
      }
      initialOutDir = next
      continue
    }

    if (arg.startsWith('--output=')) {
      initialOutDir = arg.slice('--output='.length)
      continue
    }
    if (arg.startsWith('--out-dir=')) {
      initialOutDir = arg.slice('--out-dir='.length)
      continue
    }

    if (arg.startsWith('-')) {
      return {help: false, version: false, error: `unknown option "${arg}".`}
    }

    positionals.push(arg)
  }

  if (positionals.length > 1) {
    return {help: false, version: false, error: 'expected a single url.'}
  }

  initialUrl = positionals[0]
  return {help: false, version: false, initialUrl, initialOutDir}
}
