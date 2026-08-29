# anpan

<p><br/></p>
<p align="center">
  <pre align="center">
       .,cdxkkxoc,.             
    'cdkdxxdkkdxkkkd:.          
  .dkkkkdxxdkdxkxxkkkko         
  okkkkkkkkkkkkkkOKXXXXXK0Okc.  
   dkkkkkkkkkkx0WNNK000000KXNWk 
     .kkkkkkkkOMKdc::::::::cd0Mx
               'xc:::::::::::d; 
  </pre>
</p>
<p><br/></p>

**A terminal downloader that doesn't suck.**

`anpan` is a minimal, all-in-one terminal downloader for Linux, macOS, and Windows. It combines video and audio stream extraction, multi-threaded direct file acceleration, and BitTorrent P2P transfers into a single unified CLI interface. Fast, transparent, zero-config, and just works.

## Preview

```text
          feed a link, bake a file.
youtube · x · instagram · soundcloud · torrent · and more

╭─ url / magnet / file ─────────────────────────[ bake ]─╮
│ ❯ https://... or magnet:?...                          │
╰────────────────────────────────────────────────────────╯
            ↵ bake   ·   ^s settings   ·   ^c quit
```

```text
nixos-graphical-24.11-x86_64-linux.iso

████████████████░░░░░░░░░░░░░░  54%
   2.4 MB/s  1m 15s left  · P2P (14 peers, 5 seeds)
```

## Core Stack

| Component | Software / Protocol | Role |
| :--- | :--- | :--- |
| **Media Engine** | [yt-dlp](https://github.com/yt-dlp/yt-dlp) | Video & audio stream extraction (+1,800 sites) |
| **P2P & Accelerator** | [aria2c](https://github.com/aria2/aria2) | BitTorrent/DHT engine & 16-connection multi-chunk downloader |
| **Transcoder** | [FFmpeg](https://ffmpeg.org/) | Container merging, audio extraction, thumbnail & ID3 embedding |
| **TUI Framework** | [Ink](https://github.com/vadimdemedes/ink) / React 19 | Responsive terminal UI with SGR mouse event tracking |
| **Runtime & Toolchain** | Node.js 18+ / [TypeScript 7](https://www.typescriptlang.org/) | Ultra-fast execution, sub-second typechecks, and 19ms builds |

## Features

- **All-in-One CLI** — downloads video streams, music, BitTorrent / Magnet links, and multi-threaded direct files in one tool
- **Smart target routing** — zero-config router automatically classifies inputs into Video (`yt-dlp`), P2P (`aria2c`), or Direct File without manual flags
- **BitTorrent & Magnet P2P** — fast DHT transfers with real-time peer/seed stats, ETA countdown, and immediate exit upon 100% completion
- **16-connection acceleration** — splits direct files (`.iso`, `.zip`, `.tar.gz`...) into parallel segments via `aria2c` for maximum bandwidth
- **Portion selector** — interactive resolution picker (4K down to 360p) with estimated file sizes and audio MP3 extraction
- **Time range trimming** — download specific sections by appending timestamps (e.g. `anpan "url 01:20-03:45"` or `45-90`)
- **Cover art & ID3 metadata** — automatically embeds album art, artist, and track metadata into downloaded audio
- **Playlist & Album mode** — flat-probes full playlists in ~1s with sequential numbered track naming
- **One-touch clipboard (`Tab`)** — automatically detects copied links and starts download with a single keystroke
- **Suckless terminal aesthetic** — respects terminal colors and transparency; no terminal jitter or font wrapping artifacts
- **Mouse & vi-keys support** — navigate with arrows, `j`/`k`, or click directly on UI buttons with your mouse
- **Zero configuration** — automatically fetches standalone `yt-dlp` and `ffmpeg` if not already installed

## How it works

- **Smart Target Router**: Automatically classifies incoming links into Video streams, BitTorrent P2P transfers, or Multi-threaded Direct Downloads with zero manual flags.
- **Standalone yt-dlp**: On first run, `anpan` downloads the standalone yt-dlp binary to `~/.anpan/bin` — no Python required. If you already have yt-dlp installed on PATH, it uses yours.
- **aria2c Engine**: Accelerates direct downloads with up to 32 parallel connections (`-x -s -k 1M -j`) and handles BitTorrent transfers with real-time peer/seed tracking, exiting cleanly at 100%.
- **FFmpeg Transcoder**: Merges adaptive video/audio streams and extracts audio with album artwork and ID3 tags embedded. Finds system ffmpeg or falls back to bundled `ffmpeg-static`.
- **Terminal UI**: Built with [Ink](https://github.com/vadimdemedes/ink) (React for the terminal) on TypeScript 7, featuring full SGR mouse tracking, native terminal colors, and CJK width-aware boundary calculations.

## Installation

### Prerequisites
 
- **Node.js** (>= 18.0.0)
- **aria2** (*optional, for multi-connection acceleration and BitTorrent*):
  - **Linux**: `sudo pacman -S aria2` (Arch) / `sudo apt install aria2` (Debian/Ubuntu)
  - **macOS**: `brew install aria2`
  - **Windows**: `winget install aria2` (or `scoop install aria2`)
- **ffmpeg** — *optional, bundled fallback is automatically provided*

### Global Install (NPM)

```sh
npm install -g anpan
```

Or run directly without installing:

```sh
npx anpan
```

### From Source

```sh
git clone https://github.com/YOUR_USERNAME/anpan.git ~/anpan
cd ~/anpan
npm install
npm run build
sudo npm link
```

## Usage

```sh
# Interactive prompt (with clipboard auto-detection):
anpan

# Direct video / music URL:
anpan https://youtu.be/dQw4w9WgXcQ

# Trim a specific time range:
anpan "https://youtu.be/dQw4w9WgXcQ 01:20-03:45"

# BitTorrent transfer (Magnet or .torrent link):
anpan "magnet:?xt=urn:btih:d540fc48eb12f2833163eed6421d449dd8f1ce1f&dn=NixOS"

# High-speed direct file download:
anpan https://channels.nixos.org/nixos-24.11/latest-nixos-gnome-x86_64-linux.iso
```

## Keybinds

| Key | Action |
| :--- | :--- |
| `↑` / `↓` or `j` / `k` | Navigate options / select format |
| `↵ Enter` | Submit / start download |
| `⇥ Tab` | 1-touch auto-paste and download from clipboard |
| `^s` (Ctrl+S) | Open / close Settings menu |
| `Esc` | Cancel operation / return to input screen |
| `^c` (Ctrl+C) | Exit application |
| `Mouse Click` | Click on `[ bake ]`, format items, mascot, or settings |

## Settings (`^s`)

Press `Ctrl+S` anytime to open the settings panel:

| Setting | Default | Choices | Description |
| :--- | :--- | :--- | :--- |
| **`aria2c accelerator`** | `on` | `on` / `off` | Enable multi-connection download engine |
| **`aria2c connections`** | `16` | `4`, `8`, `16`, `32` | Number of concurrent connections (`-x` / `-s`) |
| **`embed audio tags & cover`** | `on` | `on` / `off` | Embed thumbnail and metadata into audio files |
| **`default format`** | `ask` | `ask`, `best`, `1080p`, `audio` | Bypass format picker with preferred quality |
| **`save directory`** | `~/Downloads` | Presets or Custom path | Destination folder for completed downloads |

Settings are automatically saved to `~/.config/anpan/config.json`.

## A note on fair use

anpan is a personal-archiving tool. Downloading content may violate a platform's terms of service — only download what you have the right to keep, and be excellent to creators.

## License

[MIT](LICENSE)
