package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/units"
	"github.com/spf13/cobra"
)

var (
	watchOutDir      string
	watchLimit       string
	watchConnections int
	watchIntervalSec int

	watchCmd = &cobra.Command{
		Use:   "watch [folder]",
		Short: "Watch a directory and automatically download dropped torrents/magnets",
		Long: `Watch a directory (default: ~/Downloads/watch) for incoming .torrent, .magnet, or link files.

When a file is detected:
  - Download begins automatically via aria2c in the background
  - The completed file is moved to <watch_dir>/.processed/
  - A desktop notification is sent upon completion

Examples:
  anpan watch
  anpan watch ~/Torrents/watch --out-dir ~/Downloads/Media --limit 10M`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			watchDir := "~/Downloads/watch"
			if len(args) > 0 {
				watchDir = args[0]
			}
			resolvedWatchDir := units.ResolveUserPath(watchDir)

			resolvedOutDir := resolvedWatchDir
			if watchOutDir != "" {
				resolvedOutDir = units.ResolveUserPath(watchOutDir)
			}

			fmt.Println("✦ anpan watch daemon started")
			fmt.Println("  Watching:    ", resolvedWatchDir)
			fmt.Println("  Destination: ", resolvedOutDir)
			fmt.Println("  Drop .torrent, .magnet, or .txt links here to download automatically.")
			fmt.Println("  (Press Ctrl+C to stop)")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\n✦ Stopping watch daemon…")
				cancel()
			}()

			cfg := engine.WatcherConfig{
				WatchDir:       resolvedWatchDir,
				OutDir:         resolvedOutDir,
				CheckInterval:  time.Duration(watchIntervalSec) * time.Second,
				Aria2cLimit:    watchLimit,
				Connections:    watchConnections,
				OnDownloadDone: func(file string, target string) {
					fmt.Printf("✓ Finished: %s\n", file)
					system.SendNotification("anpan", fmt.Sprintf("Download complete: %s", filepath.Base(file)))
				},
				OnError: func(file string, err error) {
					fmt.Printf("✗ Failed (%s): %v\n", file, err)
				},
			}

			err := engine.StartWatcher(ctx, cfg)
			if err != nil && ctx.Err() == nil {
				return err
			}
			return nil
		},
	}
)

func init() {
	watchCmd.Flags().StringVarP(&watchOutDir, "out-dir", "o", "", "Destination directory for downloaded content")
	watchCmd.Flags().StringVarP(&watchLimit, "limit", "l", "", "Download speed limit (e.g. 5M, 500K)")
	watchCmd.Flags().IntVarP(&watchConnections, "connections", "x", 16, "Maximum aria2c connections")
	watchCmd.Flags().IntVar(&watchIntervalSec, "interval", 3, "Scan interval in seconds")

	rootCmd.AddCommand(watchCmd)
}
