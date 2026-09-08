# Troubleshooting

A practical guide to resolving common issues with external dependencies, network challenges, and configurations in `anpan`.

---

## 1. External Dependencies

`anpan` is a self-contained Go binary, but leverages standard CLI utilities for media post-processing and downloads:

| Tool | Purpose | Required? |
| :--- | :--- | :--- |
| `aria2c` | Multi-connection chunk acceleration & BitTorrent transfers | **Recommended** (Essential for torrents & fast downloads) |
| `yt-dlp` | Video/audio stream extraction from 1800+ sites | **Auto-managed** (Anpan automatically downloads standalone binary if missing) |
| `ffmpeg` | Media transcoding, container remuxing, audio tag embedding | **Recommended** (Required for format conversion & audio extraction) |

### Installing Dependencies

**Arch Linux / Manjaro:**
```sh
sudo pacman -S aria2 yt-dlp ffmpeg
```

**Debian / Ubuntu:**
```sh
sudo apt update && sudo apt install -y aria2 ffmpeg
# For the latest yt-dlp:
sudo wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp -O /usr/local/bin/yt-dlp
sudo chmod a+rx /usr/local/bin/yt-dlp
```

**macOS (Homebrew):**
```sh
brew install aria2 yt-dlp ffmpeg
```

**Windows (Winget / Scoop):**
```powershell
winget install aria2.aria2 yt-dlp.yt-dlp Gyan.FFmpeg
# or via Scoop:
scoop install aria2 yt-dlp ffmpeg
```

---

## 2. YouTube Bot Challenges & Sign-In Required

If you receive errors like:
```text
Sign in to confirm you're not a bot
Video unavailable / Sign-in required
```

### Solution: Enable Browser Cookies
1. Open settings in Anpan (`Ctrl+S`).
2. Switch to the **Video (yt-dlp)** tab.
3. Move cursor to **browser cookies** and cycle (`←` / `→` or `Space`) to select the browser where you are logged into YouTube:
   - `chrome`, `firefox`, `brave`, `edge`, or `safari`.
4. Press `Esc` to save.

---

## 3. BitTorrent & Magnet Issues

### Magnet Link Stuck on "Connecting to peers…"
- **Reason**: The magnet link may have stale trackers or your network may be blocking UDP BitTorrent traffic.
- **Solution**:
  1. Anpan automatically injects 8 reliable Tier-1 public trackers into all magnet links. If still slow, ensure your router or ISP does not throttle BitTorrent DHT traffic.
  2. Check if `aria2c` is installed: `aria2c --version`.
  3. Ensure incoming UDP traffic on ports `6881-6999` is not blocked by local firewalls (e.g. `ufw`, `iptables`, or Windows Defender).

### Privacy & VPN
BitTorrent is a peer-to-peer protocol that exposes your public IP to all connecting swarm members. Using a trustworthy VPN is recommended when downloading torrents.

---

## 4. Desktop Notifications

If desktop notifications are not appearing upon download completion:

- **Linux (Wayland & X11)**: Ensure `libnotify` / `notify-send` and a notification daemon (e.g. `dunst`, `mako`, `swaync`, `fnott`, `xfce4-notifyd`) are installed:
  ```sh
  # Test sending a notification:
  notify-send "Test" "Hello from terminal"
  ```
- **Windows**: Notifications use native PowerShell BurntToast or toast notifications. Ensure "Do Not Disturb" / Focus Assist is not silencing them.
- **macOS**: Ensure terminal permissions have notification privileges enabled in *System Settings → Notifications → Terminal / iTerm2 / Kitty*.

---

## 5. Clipboard Integration (Linux)

To use `Tab` to paste or `y` to copy magnet links in Linux terminals:
- **X11**: Ensure `xclip` or `xsel` is installed (`sudo pacman -S xclip` or `sudo apt install xclip`).
- **Wayland**: Ensure `wl-clipboard` is installed (`sudo pacman -S wl-clipboard` or `sudo apt install wl-clipboard`).

---

## 6. Resetting State, Cache & Configurations

### Reset Standalone Helper Binaries
If the managed `yt-dlp` binary becomes corrupted or outdated:
```sh
rm -rf ~/.anpan/bin ~/.cache/anpan
```
Anpan will automatically re-download a fresh copy on the next run.

### Reset Configuration to Defaults
If you want to reset all settings to factory defaults:
```sh
rm ~/.config/anpan/config.json
```
Anpan will generate a clean `config.json` with recommended defaults on the next launch.

### Complete Clean Uninstallation
```sh
anpan uninstall --purge -y
```
This removes the binary, desktop integration, config directory, and caches completely.
