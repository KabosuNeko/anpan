# CLI Reference

`anpan` provides an interactive terminal UI by default, as well as a full suite of non-interactive and scriptable CLI subcommands.

---

## Command Overview

| Command | Description |
| :--- | :--- |
| `anpan` | Launch the full interactive Bubble Tea TUI |
| `anpan [url\|magnet\|hash...]` | Directly inspect and download one or more targets |
| `anpan search <query>` | Search torrents across 7 indexers with sorting, JSON output, or interactive download |
| `anpan seed <path>` | Create a bencoded `.torrent` file, generate a magnet link, and start DHT seeding |
| `anpan watch [folder]` | Run the directory watcher daemon to auto-download dropped torrents and links |
| `anpan update` | Check for updates and automatically self-update the binary |
| `anpan uninstall` | Remove the binary and optionally purge all configuration and cache |

---

## 1. Direct Downloads & Targets

### Syntax
```sh
anpan [target...] [flags]
```

### Global Flags
| Flag | Short | Description |
| :--- | :--- | :--- |
| `--out-dir` | `-o` | Destination directory for downloaded files |
| `--input` | `-i` | Batch download targets from a text file (one target per line) |
| `--file` | `-f` | Alias for `-i / --input` |
| `--version` | `-v` | Display version information |
| `--help` | `-h` | Display help message |

### Examples
```sh
# Single media link
anpan "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# Save to specific directory
anpan -o ~/Videos "https://www.youtube.com/watch?v=dQw4w9WgXcQ"

# BitTorrent Magnet link
anpan "magnet:?xt=urn:btih:8df6e26142615621983763b729f640372cf1fc34&dn=Ubuntu"

# Raw 40-char hex or 32-char Base32 InfoHash (automatically converted to magnet)
anpan "8df6e26142615621983763b729f640372cf1fc34"

# Local .torrent file
anpan ./ubuntu-24.04-desktop-amd64.iso.torrent

# Direct media or cloud file
anpan "https://files.catbox.moe/abc123.mp4"
anpan "https://pixeldrain.com/u/xyz789"

# Timestamp trimming (downloads only the specified portion)
anpan "https://www.youtube.com/watch?v=xxx 01:20-03:45"
anpan "https://www.youtube.com/watch?v=xxx 45-90"

# Batch download multiple targets sequentially
anpan "https://youtu.be/..." "https://pixiv.net/artworks/..." "magnet:?xt=..."

# Batch download from text file
anpan -i urls.txt
```

---

## 2. Torrent Search (`anpan search`)

Search torrents concurrently across multiple open indexers: **SubsPlease**, **Nyaa**, **YTS**, **EZTV**, **The Pirate Bay**, **1337x**, and **FitGirl Repacks**.

### Syntax
```sh
anpan search <query> [flags]
```

### Flags
| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--category` | `-c` | `all` | Filter by category: `all`, `anime`, `movies`, `tv`, `games` |
| `--limit` | `-l` | `30` | Maximum number of results to retrieve |
| `--sort` | `-s` | `seeds` | Sort results by: `seeds`, `size`, `size-asc`, `peers`, `name`, `source` |
| `--json` | | `false` | Output results as structured JSON (ideal for scripts/piping) |
| `--out-dir` | `-o` | Config | Destination directory when downloading via interactive prompt |

### Examples
```sh
# Search games on FitGirl, 1337x, Nyaa, and TPB
anpan search "elden ring" --category games

# Search anime and sort by smallest size first
anpan search "frieren 1080p" --category anime --sort size-asc

# Search movies with custom result limit
anpan search "oppenheimer" --category movies --limit 15

# Export search results to JSON for scripting / jq
anpan search "ubuntu" --json | jq '.[0].magnet'

# In interactive CLI mode, select item number to download immediately:
anpan search "dune"
# > Select number to download [1-30] (or 'q' to quit, 'y <num>' to copy magnet): 1
```

---

## 3. Torrent Creation & Seeding (`anpan seed`)

Create standard bencoded `.torrent` files with SHA-1 hashing, automated piece size calculation (256KB to 4MB), Tier-1 public tracker injection, and immediate high-speed DHT seeding via `aria2c`.

### Syntax
```sh
anpan seed <file|folder> [flags]
```

### Flags
| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--comment` | | `""` | Optional comment embedded into the `.torrent` metadata |
| `--tracker` | | Tier-1 | Add custom tracker announce URLs (can be specified multiple times) |
| `--out` | `-o` | `<path>.torrent` | Custom path to save the generated `.torrent` file |
| `--limit` | `-l` | `""` | Maximum upload rate during seeding (e.g. `5M`, `500K`) |
| `--no-seed` | | `false` | Generate `.torrent` and copy magnet link only; do not start seeder process |

### Examples
```sh
# Create a .torrent and start active DHT seeding immediately
anpan seed ./my-video.mp4

# Seed an entire folder
anpan seed ~/Music/Album/ --comment "Official Bandcamp FLAC Release"

# Generate .torrent and magnet URI without launching seeder process
anpan seed ./large-dist.iso --no-seed

# Seed with a 2 MB/s upload rate cap
anpan seed ./archive.tar.gz --limit 2M
```

---

## 4. Directory Download Watcher (`anpan watch`)

Run an automated background daemon that monitors a folder for dropped `.torrent`, `.magnet`, or text link files, automatically downloading them via `aria2c` and sending desktop notifications upon completion.

### Syntax
```sh
anpan watch [folder] [flags]
```

### Flags
| Flag | Short | Default | Description |
| :--- | :--- | :--- | :--- |
| `--out-dir` | `-o` | Watch folder | Target directory where finished downloads are saved |
| `--limit` | `-l` | Config | Bandwidth speed limit (e.g. `10M`, `2M`) |
| `--connections` | `-x` | `16` | Maximum parallel aria2c connections |
| `--interval` | | `3` | Scan interval in seconds |

### Examples
```sh
# Watch default directory (~/Downloads/watch)
anpan watch

# Watch custom torrent drop folder and output to media folder
anpan watch ~/Torrents/incoming --out-dir ~/Videos/Movies --limit 10M
```

---

## 5. Maintenance Commands

### Self Update
```sh
anpan update
```
Checks GitHub Releases for a newer version, downloads the appropriate platform binary, verifies the payload, and replaces the current executable atomically.

### Uninstallation
```sh
# Remove the binary
anpan uninstall

# Remove binary and purge configuration (~/.config/anpan) and cache (~/.cache/anpan)
anpan uninstall --purge -y
```

---

## 6. Desktop & Window Manager Integrations

### Application Launchers (Rofi / dmenu / Walker / Fuzzel)
`anpan` automatically registers a desktop entry:
```text
/usr/share/applications/anpan.desktop
~/.local/share/applications/anpan.desktop
```
You can launch `anpan` directly from your application launcher or bind it to a terminal window.

### Hyprland / Sway / i3 Keybinding
Add a hotkey to quickly open `anpan` in your terminal:

**Hyprland (`~/.config/hypr/hyprland.conf`):**
```ini
# Open anpan in a floating terminal
bind = $mainMod, D, exec, kitty --class anpan-float -e anpan
windowrulev2 = float, class:^(anpan-float)$
windowrulev2 = size 1000 650, class:^(anpan-float)$
windowrulev2 = center, class:^(anpan-float)$
```

**i3 / Sway (`~/.config/i3/config`):**
```ini
bindsym $mod+Shift+d exec kitty -e anpan
for_window [app_id="anpan"] floating enable
```
