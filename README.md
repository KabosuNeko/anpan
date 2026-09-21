# anpan

<p align="center">
  <img src="https://github.com/user-attachments/assets/a0bf7c2e-ce69-42d5-a6ac-12f169972c7e" alt="Anpan Logo" style="width: 160px" />
</p>
<p align="center">
  <a href="https://github.com/KabosuNeko/anpan/releases"><img src="https://img.shields.io/github/v/release/KabosuNeko/anpan?color=d4a259&label=release" alt="GitHub release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-%3E%3D1.26.5-00ADD8.svg" alt="Go Version" /></a>
</p>

`anpan` is a cozy terminal downloader, BitTorrent engine, and media extractor written in pure Go: a Bubble Tea TUI with torrent search across 7 indexers, P2P torrent creation & seeding, a directory-watch daemon, and parallel direct download acceleration.

---

## Preview

<p align="center">
  <img src="https://github.com/user-attachments/assets/88496b86-b742-48e1-b468-5f673c7a08d3" alt="Anpan Preview" />
</p>

---

## Highlights

- **Multi-Source Torrent Search**: 7 indexers (SubsPlease, Nyaa, YTS, EZTV, The Pirate Bay, 1337x, FitGirl Repacks) with seeder filtering, sort modes, and pagination.
- **Instant Torrent Creation & Seeding**: Generate bencoded `.torrent` files and Tier-1 tracker magnet links from any local file or directory, then seed via aria2c DHT.
- **Directory Watch Daemon**: Automatically downloads `.torrent`, `.magnet`, or link files dropped into a watched folder.
- **Parallel Downloads**: Multi-connection segmented downloads (up to 32 connections) via `aria2c`.
- **Stream & Media Extraction**: Full yt-dlp integration with codec selection, audio conversion, synchronized `.lrc` lyrics, ID3 tags, and SponsorBlock.
- **Tabbed Settings**: 4-tab configuration modal (`General`, `Video`, `Audio`, `Torrent`).
- **Single Static Binary**: Zero CGo, with a built-in self-updater.

---

## Installation

**Linux / macOS:**
```sh
curl -fsSL https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/KabosuNeko/anpan/main/install.ps1 | iex
```

**Arch Linux (AUR):**
```sh
yay -S anpan-git   # or: paru -S anpan-git
```

**Build from source** (requires Go 1.26.5+):
```sh
git clone https://github.com/KabosuNeko/anpan.git
cd anpan
go build -o anpan . && sudo mv anpan /usr/local/bin/
```

---

## Quick Start

```sh
anpan                                                  # Launch the interactive TUI
anpan "https://www.youtube.com/watch?v=dQw4w9WgXcQ"   # Download a URL, magnet, or InfoHash
anpan search "elden ring" --category games             # Search torrents
anpan seed ~/Music/Album/                              # Create a .torrent and seed it
anpan watch ~/Downloads/watch                          # Auto-download dropped torrents
anpan update                                           # Self-update to the latest release
anpan uninstall --purge -y                             # Remove binary, config, and cache
```

---

## Documentation

- 🧭 [Supported Sites & Mechanisms](docs/SUPPORTED_SITES.md) — Scrapers, indexers, and routing architecture.
- ⚙️ [Configuration & Keybindings](docs/CONFIGURATION.md) — 4-tab settings, config keys, and complete TUI key reference.
- 🔧 [Troubleshooting](docs/TROUBLESHOOTING.md) — External tools (`yt-dlp`, `aria2c`, `ffmpeg`), cookies, and BitTorrent networking.

---

## License

[MIT](LICENSE)
