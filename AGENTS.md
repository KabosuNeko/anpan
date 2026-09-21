# anpan — project instructions

Pure Go terminal downloader: Bubble Tea TUI, torrent search and seeding, media extraction via yt-dlp/aria2c. Module `github.com/KabosuNeko/anpan`, Go 1.26.

## Commands

- Build: `go build ./...` (binary: `go build -o anpan .`)
- Test: `go test -count=1 ./...`
- Format: `gofmt -w .`, then `gofmt -l .` must be empty
- Vet/lint: `go vet ./...` and `go run honnef.co/go/tools/cmd/staticcheck@latest ./...`

## Layout

- `cmd/` — cobra commands (root, search, seed, watch, update, uninstall)
- `internal/core/` — target router: classifies input into torrent, search, seed, video, archive, or direct
- `internal/engine/` — extractors and download backends (yt-dlp, aria2c, site scrapers, torrent create/search)
- `internal/system/` — config JSON, history, update check, notifications, clipboard
- `internal/tui/` — Bubble Tea model, views, settings
- `internal/units/` — byte/duration/path formatting

## Constraints

- Keep `install.sh`, `install.ps1`, and `.goreleaser.yaml` consistent: release assets are version-less (`anpan-<os>-<arch>`), archives must contain `assets/*` and `LICENSE`.
- `~/.config/anpan/config.json` is user-facing; do not rename keys or change defaults without a migration path.
- Emoji in CLI/TUI output is intentional.
- Comments: godoc on exported symbols and non-obvious intent only; no narrative step comments.
- `main` must stay green: build, vet, and full tests before reporting done.
- Ask before deleting a public file, changing a CLI flag, or altering the config schema.
