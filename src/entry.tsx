import {createRequire} from 'node:module'
import {render} from 'ink'
import {AnpanApp, type Outcome} from './tui/AnpanApp.js'
import {captureFrames} from './tui/events/hitTest.js'
import {parseArgs} from './cli/options.js'
import {readClipboard} from './system/clipboard.js'
import {isLikelyTarget} from './core/domains.js'
import {checkUpdate} from './system/update.js'

const VERSION: string = createRequire(import.meta.url)('../package.json').version

const HELP = `
  anpan — feed a link, bake a file.
  (youtube · x · instagram · soundcloud · torrent · and more)

  Usage
    $ anpan [url|magnet|file] [options]

  Examples
    $ anpan https://youtu.be/dQw4w9WgXcQ
    $ anpan https://youtu.be/dQw4w9WgXcQ -o ~/Videos
    $ anpan "magnet:?xt=urn:btih:..."
    $ anpan https://example.com/nixos-minimal.iso
    $ anpan                 (prompts for input)

  Options
    -o, --output <dir>  specify download output directory
    -h, --help          show this help
    -v, --version       show version

  Downloads are saved to ~/Downloads (configure via ^s).
  Powered by yt-dlp & aria2c.
`

const args = parseArgs(process.argv.slice(2))

if (args.error) {
  console.error(`anpan: ${args.error}\nTry "anpan --help" for usage.`)
  process.exit(1)
}

if (args.help) {
  console.log(HELP)
  process.exit(0)
}

if (args.version) {
  console.log(VERSION)
  process.exit(0)
}

const initialUrl = args.initialUrl
const initialOutDir = args.initialOutDir
const isTTY = Boolean(process.stdout.isTTY)

let clipboardUrl: string | undefined
if (!initialUrl && isTTY) {
  const clipped = readClipboard().trim()
  if (clipped && !/\s/.test(clipped) && isLikelyTarget(clipped)) clipboardUrl = clipped
}

const enterAltScreen = () => process.stdout.write('\x1b[?1049h\x1b[H')
const leaveAltScreen = () => process.stdout.write('\x1b[?1006l\x1b[?1000l\x1b[?1049l')

if (isTTY) {
  enterAltScreen()
  process.on('exit', leaveAltScreen)
  for (const event of ['uncaughtException', 'unhandledRejection'] as const) {
    process.on(event, (error: unknown) => {
      leaveAltScreen()
      console.error(error)
      process.exit(1)
    })
  }
}

const updatePromise = checkUpdate(VERSION)
  .then(res => (res?.updateAvailable ? res.latestVersion : null))
  .catch(() => null)

let outcome: Outcome = {}
const {waitUntilExit} = render(
  <AnpanApp
    initialUrl={initialUrl}
    initialOutDir={initialOutDir}
    clipboardUrl={clipboardUrl}
    version={VERSION}
    onOutcome={result => (outcome = result)}
  />,
  {stdout: captureFrames(process.stdout)},
)

await waitUntilExit()

if (isTTY) leaveAltScreen()
if (outcome.filepath) {
  console.log(`done → ${outcome.filepath}`)
}

const latest = await Promise.race([
  updatePromise,
  new Promise<null>(r => setTimeout(() => r(null), 50)),
])
if (latest) {
  console.log(`\x1b[33m✦ update available:\x1b[0m ${VERSION} → \x1b[32m${latest}\x1b[0m (run: npm i -g anpan-cli)`)
}
