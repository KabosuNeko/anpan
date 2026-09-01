# Configuration & Keybindings

## TUI Keybindings

| Key | Stage | Action |
| :--- | :--- | :--- |
| `Tab` | Input | Paste URL from clipboard |
| `↑` / `↓` (`k` / `j`) | Any | Navigate list / history |
| `Enter` (`↵`) | Any | Confirm / Start download / Select |
| `Space` | Archive / Pixiv | Toggle file selection |
| `a` | Archive / Pixiv | Select all / Deselect all |
| `D` / `V` / `C` | Save prompt | Set directory to Downloads / Videos / CWD |
| `Ctrl + S` | Input / Result | Open settings modal |
| `Esc` | Sub-menu | Back / Cancel |
| `Ctrl + C` | Anywhere | Quit |

---

## Configuration File

File path: `~/.config/anpan/config.json`

```json
{
  "aria2c": true,
  "connections": 16,
  "askSaveDir": true,
  "outDir": "/home/username/Downloads",
  "preferQuality": "ask",
  "videoContainer": "mp4",
  "videoCodec": "auto",
  "audioFormat": "mp3",
  "cookiesBrowser": "none",
  "subtitles": "off",
  "subLangs": "vi,en",
  "sponsorBlock": "off",
  "embedMetadata": true,
  "writeThumbnail": false,
  "notifications": true,
  "speedLimit": "unlimited",
  "lyrics": "synced"
}
```

### Options

| Key | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `aria2c` | bool | `true` | Use aria2c for multi-connection download and BitTorrent |
| `connections` | int | `16` | Connections per download (`4`, `8`, `16`, `32`) |
| `askSaveDir` | bool | `true` | Prompt for destination before downloading |
| `outDir` | string | `~/Downloads` | Default download directory |
| `preferQuality` | string | `"ask"` | Quality selection mode (`"ask"`, `"best"`, `"1080p"`, `"audio"`) |
| `videoContainer` | string | `"mp4"` | Output video container (`"mp4"`, `"mkv"`, `"webm"`) |
| `videoCodec` | string | `"auto"` | Preferred video codec (`"auto"`, `"av1"`, `"vp9"`, `"avc"`) |
| `audioFormat` | string | `"mp3"` | Audio format (`"mp3"`, `"m4a"`, `"opus"`, `"flac"`, `"wav"`) |
| `cookiesBrowser` | string | `"none"` | Extract browser cookies (`"chrome"`, `"firefox"`, `"brave"`, `"edge"`, `"safari"`) |
| `subtitles` | string | `"off"` | Subtitle mode (`"off"`, `"embed"`, `"write"`) |
| `subLangs` | string | `"vi,en"` | Comma-separated subtitle language codes |
| `sponsorBlock` | string | `"off"` | SponsorBlock mode (`"off"`, `"remove"`, `"mark"`) |
| `embedMetadata` | bool | `true` | Embed ID3 tags and cover image |
| `writeThumbnail` | bool | `false` | Save thumbnail as standalone image file |
| `notifications` | bool | `true` | Send desktop notification on download complete |
| `speedLimit` | string | `"unlimited"` | Bandwidth limit (`"unlimited"`, `"1M"`, `"5M"`, `"10M"`, `"20M"`, `"50M"`) |
| `lyrics` | string | `"synced"` | Download synced lyrics (`"synced"`, `"off"`) |
