# anpan

<p><br/></p>
<p align="center">
  <img src="https://github.com/user-attachments/assets/a0bf7c2e-ce69-42d5-a6ac-12f169972c7e" alt="Anpan Logo" style="width: 192px" />
</p>
<p align="center">
  <a href="https://www.npmjs.com/package/anpan-cli"><img src="https://img.shields.io/npm/v/anpan-cli?color=cb904d&label=npm%20version" alt="npm version" /></a>
  <a href="https://github.com/KabosuNeko/anpan/releases"><img src="https://img.shields.io/github/v/release/KabosuNeko/anpan?color=d4a259&label=release" alt="GitHub release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
  <a href="https://nodejs.org"><img src="https://img.shields.io/badge/node-%3E%3D18-brightgreen.svg" alt="Node Version" /></a>
</p>
<p><br/></p>

`anpan` is a minimal, all-in-one terminal downloader. It combines video and audio stream extraction, multi-threaded direct file acceleration, BitTorrent P2P transfers, and creator archive scraping into a single unified CLI interface. Fast, transparent, zero-config, and just works.

## Preview

<p><br/></p>
<p align="center">
  <img src="https://github.com/user-attachments/assets/30f832d8-96c5-45bd-8cf4-d91e2b69c8e9" alt="Anpan Preview" />
</p>
<p><br/></p>

## Core Stack

| Component | Software / Protocol | Role |
| :--- | :--- | :--- |
| **Media Engine** | [yt-dlp](https://github.com/yt-dlp/yt-dlp) | Video & audio stream extraction (+1,800 sites supported) |
| **P2P & Accelerator** | [aria2c](https://github.com/aria2/aria2) | BitTorrent/DHT engine, archive batch downloader & 16-connection acceleration |
| **Transcoder** | [FFmpeg](https://ffmpeg.org/) | Adaptive stream multiplexing, audio extraction, thumbnail & ID3 embedding |
| **TUI Framework** | [Ink](https://github.com/vadimdemedes/ink) / React 19 | Responsive terminal UI with SGR mouse tracking & transparent theme |
| **Runtime & Toolchain** | Node.js 18+ / [TypeScript](https://www.typescriptlang.org/) | 100% pure TypeScript codebase, sub-second typechecks, instant execution |

## Supported Sources & Protocols

- **Streaming Video & Music**: YouTube, YouTube Music, SoundCloud, Bandcamp, Twitch, TikTok, X (Twitter), Instagram, Threads, Reddit, Facebook, Vimeo, and +1,800 other sites.
- **Creator Archive Platforms**: Kemono (`kemono.cr`, `kemono.su`), Coomer (`coomer.st`), Pawchive (`pawchive.st`, `pawchive.pw`) — scrapes and batch-downloads posts across Patreon, Fanbox, Fantia, SubscribeStar, Gumroad, Boosty, OnlyFans, and Fansly.
- **BitTorrent P2P**: Magnet links and `.torrent` files with decentralized DHT discovery, real-time peer/seed tracking, and automatic exit upon 100% completion.
- **Direct HTTP/HTTPS Files**: High-speed parallel chunk downloading for `.iso`, `.zip`, `.tar.gz`, `.mp4`, or any direct file URL.

## Key Features

- **Smart Target Routing** — zero-config router automatically classifies inputs into Video/Music (`yt-dlp`), Archive Posts (`aria2c batch`), P2P (`aria2c torrent`), or Direct Files without requiring manual CLI flags.
- **Archive Post Batch Downloader** — paste a post URL from Kemono, Coomer, or Pawchive to inspect all attached files. Download everything concurrently into an organized folder (`~/Downloads/<Post Title>/`) or select individual files.
- **Intelligent Mirror Failover** — automatically falls back to high-speed Pawchive CDN (`file.pawchive.pw`) when Kemono storage servers encounter downtime or regional routing issues.
- **BitTorrent & Magnet Transfers** — fast DHT transfers with real-time peer count, seed count, transfer speed, and ETA countdown.
- **16-Connection Acceleration** — splits direct files into parallel chunks via `aria2c` (`-x -s -k 1M -j`) to saturate available bandwidth.
- **Pristine Audio & ID3 Metadata** — extracts audio in your preferred format (`mp3`, `m4a`, `opus`, `flac`, `wav`), exposes uncompressed native Opus/AAC streams, and embeds album art and ID3 metadata tags.
- **Clean Album & Playlist Organization** — flat-probes full playlists in ~1s; automatically strips clutter prefixes (`Album -`, `EP -`, `Single -`) from folder names and UI titles.
- **Time Range Trimming** — download specific portions by appending timestamps to the link (e.g. `anpan "url 01:20-03:45"` or `45-90`).
- **Interactive Save Location Prompt** — choose save destination on the fly (`[↵] default`, `[D] ~/Downloads`, `[V] ~/Videos`, or custom path input).
- **One-Touch Clipboard (`Tab`)** — automatically inspects your clipboard on launch; start downloading with a single keystroke.
- **Zero-Manual Dependencies** — automatically fetches and manages a self-contained standalone `yt-dlp` binary (bundled with Python, Mutagen, and Brotli) and bundled `ffmpeg-static` fallback. No manual Python, pip, or dependency setup required.
- **Cross-Platform & Windows Hardened** — fully tested on Linux, macOS, and Windows with `safeReplaceBinary` atomic updating to prevent `EPERM` locks.

## Installation

### Prerequisites

- **Node.js** (>= 18.0.0)
- **aria2** (*recommended for multi-connection acceleration, archives, and BitTorrent*):
  - **Arch Linux**: `sudo pacman -S aria2`
  - **Debian / Ubuntu**: `sudo apt install aria2`
  - **Fedora**: `sudo dnf install aria2`
  - **macOS**: `brew install aria2`
  - **Windows**: `winget install aria2` (or `scoop install aria2`)
- **ffmpeg** — *optional, system binary is detected or bundled `ffmpeg-static` is automatically used*

### Global Install (NPM)

```sh
npm install -g anpan-cli
```
*(use `sudo` if required by your global npm directory permissions)*

Or run instantly without installing:

```sh
npx anpan-cli
```

### From Source

```sh
git clone https://github.com/KabosuNeko/anpan.git ~/anpan
cd ~/anpan
npm install
npm run build
sudo npm link
```

## Usage

```sh
# Launch interactive TUI (with clipboard auto-detection):
anpan

# Direct video / music stream:
anpan https://youtu.be/dQw4w9WgXcQ

# Trim a specific time range:
anpan "https://youtu.be/dQw4w9WgXcQ 01:20-03:45"

# Creator archive post (batch download all attachments):
anpan https://kemono.cr/patreon/user/49965584/post/150370074
anpan https://pawchive.st/fanbox/user/3873554/post/12509033

# BitTorrent transfer (Magnet link or .torrent):
anpan "magnet:?xt=urn:btih:d540fc48eb12f2833163eed6421d449dd8f1ce1f&dn=ArchLinux"

# High-speed parallel direct file download:
anpan https://channels.nixos.org/nixos-24.11/latest-nixos-gnome-x86_64-linux.iso

# Custom output directory:
anpan -o ~/Music https://music.youtube.com/watch?v=1yyTxvzeGuw
```

## Keybinds

| Key | Action |
| :--- | :--- |
| `↑` / `↓` or `j` / `k` | Navigate selection / portion lists |
| `↵ Enter` | Confirm selection / start download |
| `⇥ Tab` | 1-touch auto-paste and probe from clipboard |
| `^s` (Ctrl+S) | Open / close Settings menu |
| `Esc` | Cancel operation / back to input screen |
| `^c` (Ctrl+C) | Quit application |
| `Mouse Click` | Click on `[ bake ]`, options, mascot, or settings |

## Settings (`^s`)

Press `Ctrl+S` anytime to configure options. Settings are persisted to `~/.config/anpan/config.json`:

| Setting | Default | Choices | Description |
| :--- | :--- | :--- | :--- |
| **`ask save location`** | `off` | `off` / `on` | Prompt for save folder before every download |
| **`default save dir`** | `~/Downloads` | Path | Default destination directory for completed files |
| **`video format (container)`** | `mp4` | `mp4`, `mkv`, `webm` | Container format for merged video streams |
| **`audio format`** | `mp3` | `mp3`, `m4a`, `opus`, `flac`, `wav` | Audio format when converting extracted music |
| **`subtitles`** | `off` | `off`, `embed`, `write` | Download and embed or save subtitles to file |
| **`subtitle languages`** | `vi,en` | `vi,en`, `all`, `en` | Subtitle language preferences |
| **`sponsorblock`** | `off` | `off`, `remove`, `mark` | Automatically remove or mark sponsored segments |
| **`browser cookies`** | `none` | `none`, `chrome`, `firefox`, `brave`, `edge`... | Import browser session cookies for member-only media |
| **`aria2c accelerator`** | `on` | `on` / `off` | Enable multi-connection download engine |
| **`aria2c connections`** | `16` | `4`, `8`, `16`, `32` | Number of concurrent connections (`-x` / `-s`) |
| **`embed audio tags & cover`** | `on` | `on` / `off` | Embed album art, artist, and title into audio files |
| **`write thumbnail image`** | `off` | `off` / `on` | Save standalone thumbnail artwork alongside media |
| **`auto-select quality`** | `ask` | `ask`, `best`, `1080p`, `audio` | Bypass format selection with preferred quality |

## A note on fair use

`anpan` is intended for personal archiving, backups, and fair use. Please respect content creators and platform terms of service.

## License

[MIT](LICENSE)
