# anpan

```
       .,cdxkkxoc,.             
    'cdkdxxdkkdxkkkd:.          
  .dkkkkdxxdkdxkxxkkkko         
  okkkkkkkkkkkkkkOKXXXXXK0Okc.  
   dkkkkkkkkkkx0WNNK000000KXNWk 
     .kkkkkkkkOMKdc::::::::cd0Mx
               'xc:::::::::::d; 
```

feed a link, bake a file.
youtube · x · instagram · soundcloud · torrent · and more

Download videos, music, BitTorrent transfers, and direct files from YouTube,
X/Twitter, Instagram, TikTok, SoundCloud, and 1,800+ other sites — right from
your terminal. Paste a URL, magnet link, or file link, pick a resolution (or let
it stream at 16 connections), done. No popups, no fake download buttons, no
sketchy redirects.

```text
╭─ url / magnet / file ─────────────────────────[ bake ]─╮
│ ❯ https://... or magnet:?...                          │
╰────────────────────────────────────────────────────────╯
            ↵ bake   ·   ^s settings   ·   ^c quit
```

## Install

```sh
npm install -g anpan
```

Or try it without installing anything:

```sh
npx anpan
```

Requires Node 18+. Everything else (yt-dlp, ffmpeg) is fetched or bundled
automatically.

## Usage

```sh
$ anpan                                         # prompts for a url / clipboard paste
$ anpan https://youtu.be/dQw4w9WgXcQ            # straight to the format picker
$ anpan "https://youtu.be/dQw4w9WgXcQ 01:20-03:45" # trim time range
$ anpan "magnet:?xt=urn:btih:..."               # download BitTorrent transfer
$ anpan https://channels.nixos.org/nixos-24.11/latest-nixos-gnome-x86_64-linux.iso
```

anpan takes over the terminal (full-screen, centered — and restores your
scrollback on exit). Pick a format with ↑/↓ (or j/k) and hit enter. `esc`
goes back, `^c` quits. Or just use the mouse — the bake button, the format list,
and the footer hints are all clickable, and clicking the logo takes you back
home. Files are saved to `~/Downloads`, and the file path is printed to your
terminal when you're done.

Press `⇥ Tab` on launch to one-touch paste and download whatever link is currently
in your clipboard. Press `^s` or click the settings hint to configure aria2c
acceleration, connection count (4 to 32), default quality, audio metadata/cover
embedding, and output directories.

```text
nixos-graphical-24.11-x86_64-linux.iso

████████████████░░░░░░░░░░░░░░  54%
   2.4 MB/s  1m 15s left  · P2P (14 peers, 5 seeds)
```

## How it works

- **Smart Target Router**: Automatically classifies incoming links into Video streams,
  BitTorrent P2P transfers, or Multi-threaded Direct Downloads with zero manual flags.
- **Powered by yt-dlp**: On first run, anpan downloads the standalone yt-dlp binary
  to `~/.anpan/bin` — no Python required. If you already have yt-dlp installed, it
  uses yours.
- **aria2c Engine**: Accelerates direct downloads with 16 parallel connections and
  handles BitTorrent P2P transfers with real-time peer/seed tracking, exiting
  cleanly at 100%.
- **FFmpeg Transcoder**: Merges adaptive video/audio streams and extracts audio with
  album artwork and ID3 tags embedded. Finds system ffmpeg or falls back to bundled `ffmpeg-static`.
- **Terminal UI**: Built with [Ink](https://github.com/vadimdemedes/ink) (React for
  the terminal) on TypeScript 7, featuring full SGR mouse tracking, native terminal
  colors, and CJK width-aware boundary calculations.

## Development

```sh
npm install
npm run build        # bundle to dist/ with tsup (19ms)
npm run dev          # rebuild on change
npm run test         # run unit test suite (22 tests)
npm run typecheck    # check TypeScript types (0.2s)
node dist/entry.js <url>
```

To try it as a global command on your machine without publishing: `sudo npm link`,
then run `anpan` anywhere.

## A note on fair use

anpan is a personal-archiving tool. Downloading content may violate a
platform's terms of service — only download what you have the right to
keep, and be excellent to creators.

## License

[MIT](LICENSE)
