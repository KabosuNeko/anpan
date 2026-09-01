# Troubleshooting

## YouTube Bot Challenge / Sign-in prompt
Set browser cookies in settings (`Ctrl+S` -> `browser cookies` -> select `chrome`/`firefox`/etc.).

## Desktop notifications not showing on Wayland
Ensure a notification daemon is running:
- `mako` / `swaync` / `dunst`
- In Quickshell / Ringo Shell, verify `NotificationServer` is enabled.

Test notification manually:
```sh
notify-send "anpan" "Test notification"
```

## Reset standalone yt-dlp binary
```sh
rm -rf ~/.anpan/bin
```
`anpan` will download a clean binary on next run.
