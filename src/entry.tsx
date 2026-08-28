import React from 'react'
import {createRequire} from 'node:module'
import {render} from 'ink'
import {AnpanApp, type Outcome} from './tui/AnpanApp.js'
import {captureFrames} from './tui/events/hitTest.js'
import {parseArgs} from './cli/options.js'
import {readClipboard} from './system/clipboard.js'
import {isLikelyUrl} from './core/domains.js'

const VERSION: string = createRequire(import.meta.url)('../package.json').version

const HELP = `
  anpan — minimal terminal video downloader

  Usage
    $ anpan [url]

  Examples
    $ anpan https://youtu.be/dQw4w9WgXcQ
    $ anpan                 (prompts for a url)

  Options
    -h, --help      show this help
    -v, --version   show version

  Downloads are saved to ~/Downloads (configure via ^s).
  Powered by yt-dlp.
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
const isTTY = Boolean(process.stdout.isTTY)

// clipboard suggestion when no url is given
let clipboardUrl: string | undefined
if (!initialUrl && isTTY) {
  const clipped = readClipboard().trim()
  if (clipped && !/\s/.test(clipped) && isLikelyUrl(clipped)) clipboardUrl = clipped
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

let outcome: Outcome = {}
const {waitUntilExit} = render(
  <AnpanApp
    initialUrl={initialUrl}
    clipboardUrl={clipboardUrl}
    onOutcome={result => (outcome = result)}
  />,
  {stdout: captureFrames(process.stdout)},
)

await waitUntilExit()

if (isTTY) leaveAltScreen()
if (outcome.filepath) {
  console.log(`done → ${outcome.filepath}`)
}
