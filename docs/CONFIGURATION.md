# Configuration & Keybindings

`anpan` can be configured interactively through its tabbed settings modal (`Ctrl+S`) or by editing the JSON configuration file directly.

---

## 1. TUI Keybindings Reference

### Global Shortcuts
| Key | Context | Action |
| :--- | :--- | :--- |
| `Ctrl + S` | Input / Search / Results | Open tabbed settings modal |
| `Ctrl + C` | Anywhere | Immediate graceful quit / Cancel current process |
| `Esc` | Modals & Menus | Close modal, cancel prompt, or step back |

---

### Main Screen (`StageInput`)
| Key | Action |
| :--- | :--- |
| `Enter` (`↵`) | Bake input (Inspect URL, magnet, InfoHash, or search query) |
| `Ctrl + F` (`^f`) | Open Torrent Search & Browse interface |
| `Ctrl + E` (`^e`) | Open Torrent Creation & Seeding prompt |
| `Ctrl + S` (`^s`) | Open Settings modal |
| `Tab` | Paste URL / target from clipboard into input box |
| `↑` / `↓` | Recall previous target input history |

---

### Torrent Search & Browse (`StageSearch`)
The torrent search interface utilizes 100% of the terminal viewport with responsive 2-column layout:

| Key | Context | Action |
| :--- | :--- | :--- |
| `↑` / `↓` (`k` / `j`) | Results / Sidebar | Move item cursor |
| `←` / `→` (`h` / `l`) | Any | Switch focus between Sidebar, Search input, and Results |
| `Tab` / `Shift + Tab` | Any | Cycle focus between UI regions |
| `/` | Any | Instantly focus search input field |
| `Enter` (`↵`) or `d` | Results | Download selected torrent using default settings |
| `D` (Shift + D) | Results | Choose custom destination folder before downloading |
| `i` or `Space` | Results | Toggle full Inspector detail view for the selected release |
| `y` | Results | Copy magnet link of selected release to system clipboard |
| `s` | Results | Cycle sort mode: `seeds ↓` → `size ↓` → `size ↑` → `peers ↓` → `name A–Z` → `source A–Z` |
| `f` | Results | Cycle minimum seeder filter: `All` → `1+` → `5+` → `20+` seeds |
| `[` / `]` or `<` / `>` | Results | Navigate to Previous / Next page |
| `p` / `n` | Results | Navigate to Previous / Next page |
| `PgDn` / `PgUp` | Results | Jump cursor down / up by 10 items |
| `Ctrl + D` / `Ctrl + U` | Results | Jump cursor down / up by 10 items |
| `Home` / `End` | Results | Jump to the very first / last item in the list |
| `Esc` | Any | Return to main Anpan screen |

---

### Torrent Detail Inspector Modal
Opened with `i` or `Space` on any search result:

| Key | Action |
| :--- | :--- |
| `Enter` (`↵`) or `d` | Download this release |
| `D` (Shift + D) | Pick custom destination folder for this download |
| `y` | Copy full magnet URI to clipboard |
| `Esc` or `i` | Close inspector and return to search results table |

---

### Destination Folder Picker (`StageDest`)
| Key | Action |
| :--- | :--- |
| `↑` / `↓` | Navigate preset destination options |
| `D` | Quick-select standard `~/Downloads` |
| `V` | Quick-select `~/Videos` |
| `C` | Quick-select Current Working Directory |
| `e` | Focus text box to edit custom directory path |
| `Enter` (`↵`) | Confirm selected destination and start download |
| `Esc` | Cancel and return to previous screen |

---

### Tabbed Settings Modal (`StageSettings`)
Press `Ctrl+S` from the main screen or results to open the 4-tab compact configuration interface:

| Key | Action |
| :--- | :--- |
| `Tab` / `Shift + Tab` | Switch to Next / Previous Settings Tab |
| `[` / `]` | Switch to Previous / Next Settings Tab |
| `1`, `2`, `3`, `4` | Jump directly to Tab 1 (`General`), 2 (`Video`), 3 (`Audio`), or 4 (`Torrent`) |
| `↑` / `↓` | Move cursor between setting items in the current tab |
| `←` / `→` or `Space` | Cycle through available options for the selected setting |
| `Enter` (`↵`) | Edit folder path or toggle boolean setting |
| `Esc` | Save all changes and close settings |

---

### Multi-File Post Selector (Pixiv / Kemono / Coomer / Archive.org)
| Key | Action |
| :--- | :--- |
| `↑` / `↓` | Move cursor through the list of attachments |
| `Space` | Toggle checkbox for the selected file |
| `a` | Select All / Deselect All files |
| `Enter` (`↵`) | Confirm selection and download chosen files in parallel |
| `Esc` | Cancel and return |

---

## 2. Configuration File Schema

Path: `~/.config/anpan/config.json`

```json
{
  "askSaveDir": true,
  "outDir": "/home/user/Downloads",
  "colorTheme": "bakery",
  "notifications": true,
  "autoPaste": false,
  "speedLimit": "unlimited",

  "videoContainer": "mp4",
  "videoCodec": "auto",
  "preferQuality": "ask",
  "subtitles": "off",
  "subLangs": "vi,en",
  "sponsorBlock": "off",
  "cookiesBrowser": "none",

  "audioFormat": "mp3",
  "embedMetadata": true,
  "lyrics": "synced",
  "writeThumbnail": false,

  "aria2c": true,
  "connections": 16,
  "torrentSeedRatio": "off",
  "defaultSearchCat": "all",
  "defaultSearchSort": "seeds"
}
```

---

## 3. Configuration Keys & Options

### Tab 1: General Settings
| Key | Type | Default | Options | Description |
| :--- | :--- | :--- | :--- | :--- |
| `askSaveDir` | bool | `true` | `true`, `false` | Prompt for destination directory before each download |
| `outDir` | string | `~/Downloads` | any path | Default directory where downloaded files are saved |
| `colorTheme` | string | `"bakery"` | `"bakery"`, `"terminal"` | Color theme: cozy bakery palette or 16-color ANSI terminal mode |
| `notifications` | bool | `true` | `true`, `false` | Send desktop notification when downloads complete |
| `autoPaste` | bool | `false` | `true`, `false` | Automatically paste clipboard content when launching Anpan |
| `speedLimit` | string | `"unlimited"` | `"unlimited"`, `"1M"`, `"5M"`, `"10M"`, `"20M"`, `"50M"` | Bandwidth limit applied to aria2c and yt-dlp |

---

### Tab 2: Video Settings (`yt-dlp`)
| Key | Type | Default | Options | Description |
| :--- | :--- | :--- | :--- | :--- |
| `videoContainer` | string | `"mp4"` | `"mp4"`, `"mkv"`, `"webm"` | Output video container format |
| `videoCodec` | string | `"auto"` | `"auto"`, `"av1"`, `"vp9"`, `"avc"` | Preferred video codec encoding |
| `preferQuality` | string | `"ask"` | `"ask"`, `"best"`, `"1080p"`, `"audio"` | Auto-select resolution or prompt for stream quality |
| `subtitles` | string | `"off"` | `"off"`, `"embed"`, `"write"` | Subtitle mode (embed inside container or write standalone `.srt`/`.vtt`) |
| `subLangs` | string | `"vi,en"` | `"vi,en"`, `"all"`, `"en"` | Preferred subtitle language codes |
| `sponsorBlock` | string | `"off"` | `"off"`, `"remove"`, `"mark"` | Auto-skip sponsored segments using SponsorBlock |
| `cookiesBrowser` | string | `"none"` | `"none"`, `"chrome"`, `"firefox"`, `"brave"`, `"edge"`, `"safari"` | Extract browser session cookies to bypass YouTube bot detection |

---

### Tab 3: Audio & Music
| Key | Type | Default | Options | Description |
| :--- | :--- | :--- | :--- | :--- |
| `audioFormat` | string | `"mp3"` | `"mp3"`, `"m4a"`, `"opus"`, `"flac"`, `"wav"` | Output audio encoding format |
| `embedMetadata` | bool | `true` | `true`, `false` | Embed ID3 tags, artist, album, track number, and cover art |
| `lyrics` | string | `"synced"` | `"synced"`, `"off"` | Download synchronized time-stamped `.lrc` lyrics |
| `writeThumbnail` | bool | `false` | `true`, `false` | Save high-resolution album/video cover art as a standalone image |

---

### Tab 4: BitTorrent & `aria2c`
| Key | Type | Default | Options | Description |
| :--- | :--- | :--- | :--- | :--- |
| `aria2c` | bool | `true` | `true`, `false` | Use aria2c for parallel chunk downloads and BitTorrent engine |
| `connections` | int | `16` | `4`, `8`, `16`, `32` | Maximum concurrent connections per direct download |
| `torrentSeedRatio` | string | `"off"` | `"off"`, `"1.0"`, `"2.0"`, `"unlimited"` | Seeding stop ratio after torrent completion |
| `defaultSearchCat` | string | `"all"` | `"all"`, `"anime"`, `"movies"`, `"tv"`, `"games"` | Default category active when opening torrent search |
| `defaultSearchSort` | string | `"seeds"` | `"seeds"`, `"size"`, `"size-asc"`, `"peers"`, `"name"`, `"source"` | Default sorting mode for torrent search results |
