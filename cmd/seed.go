package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/units"
	"github.com/spf13/cobra"
)

var (
	seedComment  string
	seedTrackers []string
	seedOutPath  string
	seedLimit    string
	seedOnlyMeta bool

	seedCmd = &cobra.Command{
		Use:   "seed <file|folder>",
		Short: "Create a .torrent and start BitTorrent P2P seeding via DHT",
		Long: `Create a bencoded .torrent file for any file or directory and seed it immediately via aria2c.

Features:
  - Automatic piece length calculation (256KB to 4MB)
  - Public Tier-1 tracker injection
  - Generates shareable Magnet URI and copies it to clipboard
  - Starts high-efficiency DHT seeding via aria2c

Examples:
  anpan seed ./my-video.mp4
  anpan seed ~/Music/Album/ --comment "Official release"
  anpan seed ./dist/ --no-seed (generates .torrent and magnet only)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			sourcePath := units.ResolveUserPath(args[0])

			fi, err := os.Stat(sourcePath)
			if err != nil {
				return fmt.Errorf("source path not found: %w", err)
			}

			fmt.Printf("✦ Creating torrent for %s (%s)…\n", fi.Name(), units.FormatBytes(float64(fi.Size())))

			opts := &engine.TorrentCreateOptions{
				Comment:  seedComment,
				Trackers: seedTrackers,
			}

			created, err := engine.CreateTorrent(sourcePath, seedOutPath, opts)
			if err != nil {
				return fmt.Errorf("failed to create torrent: %w", err)
			}

			fmt.Println()
			fmt.Println("✓ Torrent created successfully!")
			fmt.Println("  File:      ", created.TorrentPath)
			fmt.Println("  InfoHash:  ", created.InfoHash)
			fmt.Println("  Total Size:", units.FormatBytes(float64(created.TotalBytes)))
			fmt.Println("  Pieces:    ", fmt.Sprintf("%d × %s", created.PiecesCount, units.FormatBytes(float64(created.PieceLength))))
			fmt.Println()
			fmt.Println("  Magnet:")
			fmt.Println(" ", created.Magnet)
			fmt.Println()

			_ = system.WriteClipboard(created.Magnet)
			fmt.Println("✓ Copied magnet link to clipboard!")

			if seedOnlyMeta {
				return nil
			}

			ariaBin, err := engine.FindAria2c()
			if err != nil {
				fmt.Println("⚠ aria2c not found; skipping active DHT seeding.")
				return nil
			}

			fmt.Println("✦ Starting active P2P seeding with aria2c… (Press Ctrl+C to stop)")

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
			go func() {
				<-sigChan
				fmt.Println("\n✦ Stopping seeder…")
				cancel()
			}()

			seedDir := filepath.Dir(sourcePath)
			_, bakeErr := engine.BakeTorrentSeed(ctx, engine.TorrentSeedOptions{
				Aria2cBin:   ariaBin,
				TorrentPath: created.TorrentPath,
				OutputDir:   seedDir,
				UploadLimit: seedLimit,
			}, engine.BakeHandlers{})

			if bakeErr != nil && ctx.Err() == nil {
				return bakeErr
			}

			fmt.Println("✓ Seeding completed.")
			return nil
		},
	}
)

func init() {
	seedCmd.Flags().StringVar(&seedComment, "comment", "", "Optional comment to include in the torrent metadata")
	seedCmd.Flags().StringSliceVar(&seedTrackers, "tracker", nil, "Custom tracker URLs (defaults to Tier-1 public trackers)")
	seedCmd.Flags().StringVarP(&seedOutPath, "out", "o", "", "Destination path for generated .torrent file")
	seedCmd.Flags().StringVarP(&seedLimit, "limit", "l", "", "Max upload rate (e.g. 5M, 500K)")
	seedCmd.Flags().BoolVar(&seedOnlyMeta, "no-seed", false, "Generate .torrent and copy magnet only; do not start seeder process")

	rootCmd.AddCommand(seedCmd)
}
