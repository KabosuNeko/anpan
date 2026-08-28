type ParsedArgs =
  | {help: true; version: false; error?: undefined; initialUrl?: undefined}
  | {help: false; version: true; error?: undefined; initialUrl?: undefined}
  | {help: false; version: false; error: string; initialUrl?: undefined}
  | {help: false; version: false; error?: undefined; initialUrl?: string}

export function parseArgs(argv: string[]): ParsedArgs {
  let initialUrl: string | undefined
  const positionals: string[] = []

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i]!
    if (arg === '-h' || arg === '--help') return {help: true, version: false}
    if (arg === '-v' || arg === '--version') return {help: false, version: true}

    if (arg.startsWith('-')) {
      return {help: false, version: false, error: `unknown option "${arg}".`}
    }

    positionals.push(arg)
  }

  if (positionals.length > 1) {
    return {help: false, version: false, error: 'expected a single url.'}
  }

  initialUrl = positionals[0]
  return {help: false, version: false, initialUrl}
}
