# anpan

> minimal terminal video downloader

Download videos from YouTube, X/Twitter, Instagram, Threads, TikTok, and 1,800+ other sites — right from your terminal.

```
█▀█ █▀▄█ █▀█ █▀█ █▀▄█
█▀█ █  █ █▀▀ █▀█ █  █
▀ ▀ ▀  ▀ ▀   ▀ ▀ ▀  ▀
```

## Features

- **Suckless & Minimal**: Native terminal styling that respects your terminal colors and transparency. No bloated emojis or AI slop.
- **Portion Selector**: Interactive format picker displaying resolutions (2160p down to 360p) with estimated file sizes and audio MP3 extraction.
- **aria2c Acceleration**: Built-in multi-connection downloader support with configurable thread count (4, 8, 16, 32).
- **Settings Panel (`^s`)**: In-app configuration menu to toggle aria2c, adjust connections, set default quality, and view output paths.
- **Mouse & Keyboard Friendly**: Navigate with arrows, vi-keys (`j`/`k`), or click directly with your mouse.
- **Zero Config**: Automatically fetches standalone `yt-dlp` to `~/.anpan/bin` if not already installed.

## Installation

```bash
npm install -g anpan
```

Or run directly without installing:

```bash
npx anpan
```

Requires Node.js 18+.

## Usage

```bash
$ anpan https://youtu.be/dQw4w9WgXcQ    # jump straight to format picker
$ anpan                                 # interactive prompt with clipboard paste
```

### Shortcuts

- `↑` / `↓` or `j` / `k`: Navigate / choose format
- `↵ Enter`: Submit / download
- `^s`: Open / close settings
- `Esc`: Cancel / go back
- `^c`: Quit
- Mouse: Click buttons, options, or settings directly

Files are saved to `~/Downloads` (configurable via `^s`).

## License

[MIT](LICENSE)
