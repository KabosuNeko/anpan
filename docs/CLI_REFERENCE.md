# CLI Reference

## Usage

```sh
# Interactive TUI
anpan

# Direct target
anpan "https://www.youtube.com/watch?v=WtaKxxZGCKc"

# Download to custom directory
anpan -o ~/Music "https://soundcloud.com/artist/track"

# Timestamp trimming
anpan "https://www.youtube.com/watch?v=xxx 01:20-03:45"
anpan "https://www.youtube.com/watch?v=xxx 45-90"

# Batch download from text file
anpan -i urls.txt
anpan -f list.txt

# Batch download multiple URLs
anpan "https://youtu.be/..." "https://pixiv.net/artworks/..." "magnet:?xt=..."

# Self-update
anpan update

# Uninstall
anpan uninstall
anpan uninstall --purge -y
```

---

## Desktop Integrations

### Niri
`~/.config/niri/config.kdl`:
```kdl
binds {
    Mod+D { spawn "alacritty" "-e" "anpan"; }
}
```

### Hyprland
`~/.config/hypr/hyprland.conf`:
```ini
bind = $mainMod, D, exec, kitty --class anpan-float -e anpan
windowrulev2 = float, class:^(anpan-float)$
windowrulev2 = size 850 500, class:^(anpan-float)$
```

### Rofi / App Launcher
`anpan` installs `/usr/share/applications/anpan.desktop` (or `~/.local/share/applications/anpan.desktop`), accessible by all freedesktop app launchers.
