# anpan

<p align="center">
  <img src="https://github.com/user-attachments/assets/a0bf7c2e-ce69-42d5-a6ac-12f169972c7e" alt="Anpan Logo" style="width: 160px" />
</p>
<p align="center">
  <a href="https://github.com/KabosuNeko/anpan/releases"><img src="https://img.shields.io/github/v/release/KabosuNeko/anpan?color=d4a259&label=release" alt="GitHub release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-%3E%3D1.24-00ADD8.svg" alt="Go Version" /></a>
</p>

`anpan` is a cozy, high-performance terminal downloader, BitTorrent engine, and media extractor written in pure Go. It features a responsive Bubble Tea TUI, multi-source torrent search across 7 indexers, instant P2P torrent creation & seeding, automated directory watching, and parallel direct download acceleration.

---

## Preview

<p align="center">
  <img src="https://github.com/user-attachments/assets/88496b86-b742-48e1-b468-5f673c7a08d3" alt="Anpan Preview" />
</p>

---

## Highlights

- **🍞 Signature Warm Bakery Aesthetic**: Handcrafted cozy palette (`bakery`) with a minimalist 16-color ANSI mode (`terminal`) for custom themes.
- **🔍 Multi-Source Torrent Search**: Search and browse across 7 major indexers (SubsPlease, Nyaa, YTS, EZTV, The Pirate Bay, 1337x, FitGirl Repacks) with full-viewport responsive layout, seeder filtering, sort modes, and multi-page pagination.
- **🌱 Instant Torrent Creation & Seeding**: Generate bencoded `.torrent` files and shareable Tier-1 tracker magnet links directly from any local file or directory, and seed via aria2c DHT.
- **👁️ Directory Watch Daemon**: Background watcher that automatically downloads incoming `.torrent`, `.magnet`, or link files dropped into a folder.
- **⚡ Parallel Download Acceleration**: Multi-connection segmented downloads (up to 32 connections) for cloud hosts, imageboards, and archives via `aria2c`.
- **🎥 Stream & Media Extraction**: Full yt-dlp integration with codec selection (AV1, VP9, AVC), audio conversion, synchronized `.lrc` lyrics, ID3 tags, and SponsorBlock.
- **📑 Tabbed Ergonomic Settings**: 4-tab compact configuration modal (`General`, `Video`, `Audio`, `Torrent`) that never overflows your screen.
- **📦 Zero CGo & Self-Contained**: Single static Go binary with built-in updater.

---

## Supported Sources

| Category | Sources | Features |
| :--- | :--- | :--- |
| **BitTorrent Search** | SubsPlease, Nyaa, YTS, EZTV, The Pirate Bay, 1337x, FitGirl Repacks | Query search, category browse (Anime, Movies, TV, Games), sorting, inspector, pagination |
| **BitTorrent P2P** | Magnet links, `.torrent` files, bare 40-hex / 32-base32 InfoHashes | DHT discovery, tracker enrichment, seeding, automated folder watcher |
| **Video & Audio** | YouTube, SoundCloud, TikTok, X (Twitter), Twitch, Bilibili, +1800 sites | Stream extraction via yt-dlp, codec selection, chapters, subtitles, synced lyrics |
| **Art & Illustration** | Pixiv, Imgur Albums, Yande.re, Konachan, Safebooru, Gelbooru | Multi-page gallery extraction, original resolution images, batch downloading |
| **Archive Posts & Libraries** | Kemono, Coomer, Pawchive, Internet Archive (archive.org) | Multi-attachment extraction with mirror fallback, digital library items, ISOs |
| **Cloud & Direct** | Google Drive, MediaFire, Pixeldrain (files & lists), Catbox, Litterbox, direct HTTP/HTTPS | 16-32 connection parallel chunk acceleration via aria2c |

See [docs/SUPPORTED_SITES.md](docs/SUPPORTED_SITES.md) for full routing architecture and URL patterns.

---

## Installation

### Automated Script Installer (Recommended)

**Linux / macOS:**
```sh
curl -fsSL https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.ps1 | iex
```

### Arch Linux (AUR)

```sh
yay -S anpan-git
# or
paru -S anpan-git
```

### Build from Source

Requirements: Go 1.24+

```sh
git clone https://github.com/KabosuNeko/anpan.git
cd anpan
go build -o anpan .
sudo mv anpan /usr/local/bin/
```

---

## Quick Start

```sh
# Launch interactive TUI
anpan

# Download any media URL, magnet link, or InfoHash directly
anpan "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
anpan "magnet:?xt=urn:btih:..."
anpan "44917454083a216fe8b15d0259b665dfb119a6d8"

# Search torrents across indexers
anpan search "elden ring" --category games
anpan search "frieren" --sort size-asc --limit 10

# Create a .torrent and start seeding via DHT
anpan seed ~/Music/Album/
anpan seed ./large-archive.zip --no-seed

# Run the automated directory download watcher
anpan watch ~/Downloads/watch --out-dir ~/Downloads/Media

# Batch download from file
anpan -i links.txt
```

---

## Updates & Removal

```sh
# Self-update to the latest release
anpan update

# Uninstall cleanly
anpan uninstall

# Uninstall and purge all configs & cache
anpan uninstall --purge -y
```

---

## Documentation

- 🧭 [Supported Sites & Mechanisms](docs/SUPPORTED_SITES.md) — Scrapers, indexers, and routing architecture.
- ⚙️ [Configuration & Keybindings](docs/CONFIGURATION.md) — 4-tab settings, config schema, and complete TUI key reference.
- 💻 [CLI Reference](docs/CLI_REFERENCE.md) — Subcommands (`search`, `seed`, `watch`), flags, and window manager integration.
- 🔧 [Troubleshooting](docs/TROUBLESHOOTING.md) — External tools (`yt-dlp`, `aria2c`, `ffmpeg`), cookies, and BitTorrent networking.

---

## License

[MIT](LICENSE) © [KabosuNeko](https://github.com/KabosuNeko)
