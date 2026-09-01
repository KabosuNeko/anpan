# anpan

<p align="center">
  <img src="https://github.com/user-attachments/assets/a0bf7c2e-ce69-42d5-a6ac-12f169972c7e" alt="Anpan Logo" style="width: 160px" />
</p>
<p align="center">
  <a href="https://github.com/KabosuNeko/anpan/releases"><img src="https://img.shields.io/github/v/release/KabosuNeko/anpan?color=d4a259&label=release" alt="GitHub release" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License" /></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/go-%3E%3D1.24-00ADD8.svg" alt="Go Version" /></a>
</p>

`anpan` is a terminal downloader written in Go. It handles video and audio stream extraction, multi-threaded direct file acceleration, and BitTorrent transfers through a single Bubble Tea interface.

## Preview

<p align="center">
  <img src="https://github.com/user-attachments/assets/30f832d8-96c5-45bd-8cf4-d91e2b69c8e9" alt="Anpan Preview" />
</p>

## Supported Sources

| Category | Sources | Notes |
| :--- | :--- | :--- |
| **Video & Audio** | YouTube, SoundCloud, TikTok, X (Twitter), Twitch, Bilibili, +1800 sites | Stream extraction via yt-dlp, AV1/VP9/AVC codecs, audio conversion, metadata & chapters |
| **Art & Illustration** | Pixiv, Imgur Albums, Yande.re, Konachan, Safebooru, Gelbooru | Multi-page post & album extraction, original resolution images |
| **Archive Posts & Libraries** | Kemono, Coomer, Pawchive, Internet Archive (archive.org) | Multi-attachment extraction with mirror fallback, digital library items |
| **Cloud & Direct** | Google Drive, MediaFire, Pixeldrain (files & lists), Catbox, Litterbox, direct HTTP/HTTPS | 16-connection parallel chunk download via aria2c |
| **P2P** | Magnet links, `.torrent` files | BitTorrent transfer via aria2c |

See [docs/SUPPORTED_SITES.md](docs/SUPPORTED_SITES.md) for URL schemes and backend routing details.

## Installation

### Script installer

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

### From source

```sh
git clone https://github.com/KabosuNeko/anpan.git
cd anpan
go build -o anpan .
sudo mv anpan /usr/local/bin/
```

## Update

```sh
anpan update
```

## Uninstall

```sh
anpan uninstall
# or remove config and cache completely:
anpan uninstall --purge -y
```

## Documentation

- [Supported Sites & Mechanisms](docs/SUPPORTED_SITES.md) — Routing logic and URL formats.
- [Configuration & Keybindings](docs/CONFIGURATION.md) — Config keys and TUI keyboard shortcuts.
- [CLI Reference](docs/CLI_REFERENCE.md) — CLI flags, timestamp trimming, and WM keybindings.
- [Troubleshooting](docs/TROUBLESHOOTING.md) — Cookies, notifications, and dependencies.

## License

[MIT](LICENSE) © [KabosuNeko](https://github.com/KabosuNeko)
