# anpan

<p><br/></p>
<p align="center">
  <img src="https://github.com/user-attachments/assets/a0bf7c2e-ce69-42d5-a6ac-12f169972c7e" alt="Anpan Logo" style="width: 192px" />
</p>
<p align="center">
  <a href="https://github.com/KabosuNeko/anpan/releases"><img src="https://img.shields.io/github/v/release/KabosuNeko/anpan?color=d4a259&label=release" alt="GitHub release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-%3E%3D1.24-00ADD8.svg" alt="Go Version" /></a>
</p>
<p><br/></p>

`anpan` is a minimal, all-in-one terminal downloader written in **Go**. It combines video and audio stream extraction, multi-threaded direct file acceleration, and BitTorrent P2P transfers into a single unified CLI interface with a responsive Bubble Tea TUI. Fast, transparent, zero-config, and just works.

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
| **P2P & Accelerator** | [aria2c](https://github.com/aria2/aria2) | BitTorrent/DHT engine & 16-connection acceleration |
| **Transcoder** | [FFmpeg](https://ffmpeg.org/) | Adaptive stream multiplexing, audio extraction, thumbnail & ID3 embedding |
| **CLI & TUI Framework** | [Cobra](https://github.com/spf13/cobra) / [Bubble Tea](https://charm.land/bubbletea/v2) / [Lip Gloss](https://charm.land/lipgloss/v2) | Modern Elm-architecture terminal UI with Lip Gloss styling |
| **Language & Runtime** | [Go](https://go.dev/) 1.24+ | Single self-contained static binary, zero runtime dependencies |

## Supported Sources & Protocols

- **Streaming Video & Music**: YouTube, YouTube Music, SoundCloud, Bandcamp, Twitch, TikTok, X (Twitter), Instagram, Threads, Reddit, Facebook, Vimeo, and +1,800 other sites.
- **BitTorrent P2P**: Magnet links and `.torrent` files with decentralized DHT discovery, real-time peer/seed tracking, and automatic exit upon 100% completion.
- **Direct HTTP/HTTPS Files**: High-speed parallel chunk downloading for `.iso`, `.zip`, `.tar.gz`, `.mp4`, or any direct file URL.

## Key Features

- **Smart Target Routing** — zero-config router automatically classifies inputs into Video/Music (`yt-dlp`), P2P (`aria2c torrent`), or Direct Files without requiring manual CLI flags.
- **BitTorrent & Magnet Transfers** — fast DHT transfers with real-time peer count, seed count, transfer speed, and ETA countdown.
- **16-Connection Acceleration** — splits direct files into parallel chunks via `aria2c` (`-x -s -k 1M -j`) to saturate available bandwidth.
- **Pristine Audio & ID3 Metadata** — extracts audio in your preferred format (`mp3`, `m4a`, `opus`, `flac`, `wav`), exposes uncompressed native Opus/AAC streams, and embeds album art and ID3 metadata tags.
- **Clean Album & Playlist Organization** — flat-probes full playlists in ~1s; automatically strips clutter prefixes (`Album -`, `EP -`, `Single -`) from folder names and UI titles.
- **Time Range Trimming** — download specific portions by appending timestamps to the link (e.g. `anpan "url 01:20-03:45"` or `45-90`).
- **Interactive Save Location Prompt** — choose save destination on the fly (`[↵] default`, custom path input).
- **One-Touch Clipboard (`Tab`)** — automatically inspects your clipboard on launch; start downloading with a single keystroke.
- **Zero-Manual Dependencies** — automatically fetches and manages a self-contained standalone `yt-dlp` binary (bundled with Python, Mutagen, and Brotli) to `~/.anpan/bin`.
- **Self-Updating** — non-blocking check on exit or run `anpan update` to instantly update in place.

## Installation

### Standalone Installer *(Recommended)*

**macOS / Linux:**
```sh
curl -fsSL https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.ps1 | iex
```

---

### Arch Linux (AUR)

For Arch Linux / Manjaro / EndeavourOS users:

```sh
yay -S anpan-git
# or
paru -S anpan-git
```

---

### From Source (Go 1.24+)

```sh
git clone https://github.com/KabosuNeko/anpan.git ~/anpan
cd ~/anpan
go build -o anpan .
sudo mv anpan /usr/local/bin/
```

---

## Update

- **Standalone installation (Recommended)**:
  Run the built-in update command anytime:
  ```sh
  anpan update
  ```
  *(Or re-run the install script)*.

- **AUR installation**:
  Update along with your system via your AUR helper:
  ```sh
  yay -Syu
  # or
  paru -Syu
  ```

## Prerequisites

- **aria2** (*recommended for multi-connection acceleration and BitTorrent*):
  - **Arch Linux**: `sudo pacman -S aria2`
  - **Debian / Ubuntu**: `sudo apt install aria2`
  - **Fedora**: `sudo dnf install aria2`
  - **macOS**: `brew install aria2`
  - **Windows**: `winget install aria2` (or `scoop install aria2`)
- **ffmpeg** — *optional, system binary is detected or local fallback in `~/.anpan/bin`*

## Usage

```sh
# Launch interactive TUI (with clipboard auto-detection):
anpan

# Direct video / music stream:
anpan https://youtu.be/dQw4w9WgXcQ

# Trim a specific time range:
anpan "https://youtu.be/dQw4w9WgXcQ 01:20-03:45"

# BitTorrent transfer (Magnet link or .torrent):
anpan "magnet:?xt=urn:btih:d540fc48eb12f2833163eed6421d449dd8f1ce1f&dn=ArchLinux"

# High-speed parallel direct file download:
anpan https://channels.nixos.org/nixos-24.11/latest-nixos-gnome-x86_64-linux.iso

# Custom output directory:
anpan -o ~/Music https://music.youtube.com/watch?v=1yyTxvzeGuw

# Update to latest version:
anpan update
```

## Keybinds

| Key | Stage | Action |
| :--- | :--- | :--- |
| `↵` (Enter) | All | Confirm / Bake / Download / Retry |
| `Tab` | Input | Paste URL from system clipboard |
| `^s` (Ctrl+S) | Input / Baked | Open quick settings modal |
| `↑` / `↓` (`k` / `j`) | Select / Settings | Navigate portions or settings |
| `←` / `→` (`h` / `l`) | Settings | Adjust setting values |
| `Esc` | Any | Cancel / Back |
| `^c` (Ctrl+C) | Any | Quit application immediately |

## License

[MIT](LICENSE) © [KabosuNeko](https://github.com/KabosuNeko)
