package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/KabosuNeko/anpan/internal/engine"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/units"
	"github.com/spf13/cobra"
)

var (
	searchCategory string
	searchLimit    int
	searchSort     string
	searchJSON     bool

	searchCmd = &cobra.Command{
		Use:   "search <query>",
		Short: "Search torrents across multiple indexers (SubsPlease, Nyaa, YTS, EZTV, PirateBay, 1337x, FitGirl)",
		Long: `Search torrents across multiple indexers and download or copy magnets directly.

Supported sources:
  - SubsPlease & Nyaa (Anime & Japanese Games)
  - YTS (Movies)
  - EZTV (TV Shows)
  - The Pirate Bay (General releases & PC/Console Games)
  - 1337x (Movies, TV Shows & Games)
  - FitGirl Repacks (Verified PC Game Repacks)

Examples:
  anpan search "frieren" --sort size-asc
  anpan search "elden ring" --category games
  anpan search "dune 2" --category movies
  anpan search "breaking bad" --category tv
  anpan search "ubuntu" --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			ctx := context.Background()

			opts := &engine.SearchOptions{
				Category: searchCategory,
				Limit:    searchLimit,
				SortBy:   searchSort,
			}

			if !searchJSON {
				fmt.Printf("✦ searching torrent sources for %q…\n", query)
			}

			results, err := engine.SearchTorrents(ctx, query, opts)
			if err != nil {
				return err
			}

			if searchJSON {
				data, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
				return nil
			}

			if len(results) == 0 {
				fmt.Println("No torrents found for your search query.")
				return nil
			}

			fmt.Println()
			fmt.Printf(" %-4s %-48s %-12s %-10s %-8s %s\n", "#", "TITLE", "SIZE", "SEEDS", "LEECHS", "SOURCE")
			fmt.Println(strings.Repeat("─", 94))

			for i, r := range results {
				sz := units.FormatBytes(float64(r.SizeBytes))
				if sz == "" {
					sz = "-"
				}
				truncTitle := units.Truncate(r.Title, 46)
				fmt.Printf(" %-4d %-48s %-12s ▲ %-8d ▼ %-6d %s\n",
					i+1, truncTitle, sz, r.Seeders, r.Leechers, r.Source)
			}
			fmt.Println(strings.Repeat("─", 94))
			fmt.Println()

			// Interactive download prompt if terminal
			fmt.Print("Select number to download [1-", len(results), "] (or 'q' to quit, 'y <num>' to copy magnet): ")
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)

			if input == "" || strings.ToLower(input) == "q" {
				return nil
			}

			if strings.HasPrefix(strings.ToLower(input), "y ") {
				numStr := strings.TrimSpace(input[2:])
				num, err := strconv.Atoi(numStr)
				if err == nil && num >= 1 && num <= len(results) {
					chosen := results[num-1]
					_ = system.WriteClipboard(chosen.Magnet)
					fmt.Println("✓ Copied magnet link to clipboard!")
					return nil
				}
			}

			num, err := strconv.Atoi(input)
			if err != nil || num < 1 || num > len(results) {
				fmt.Println("Invalid selection.")
				return nil
			}

			chosen := results[num-1]
			fmt.Printf("\n✦ Downloading: %s\n", chosen.Title)

			ariaBin, err := engine.FindAria2c()
			if err != nil {
				return fmt.Errorf("aria2c is required for BitTorrent download: %w", err)
			}

			cfg := system.LoadConfig()
			dest := cfg.OutDir
			if outputDir != "" {
				dest = units.ResolveUserPath(outputDir)
			}

			_, bakeErr := engine.BakeTorrentDownload(ctx, engine.TorrentDownloadOptions{
				Aria2cBin:  ariaBin,
				Target:     chosen.Magnet,
				OutputDir:  dest,
				SpeedLimit: cfg.SpeedLimit,
				SeedRatio:  cfg.TorrentSeedRatio,
			}, engine.BakeHandlers{})

			return bakeErr
		},
	}
)

func init() {
	searchCmd.Flags().StringVarP(&searchCategory, "category", "c", "all", "Filter by category: all, anime, movies, tv, games")
	searchCmd.Flags().IntVarP(&searchLimit, "limit", "l", 30, "Maximum number of results to return")
	searchCmd.Flags().StringVarP(&searchSort, "sort", "s", "seeds", "Sort results by: seeds, size, size-asc, peers, name, source")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output results as formatted JSON")
	searchCmd.Flags().StringVarP(&outputDir, "out-dir", "o", "", "Destination directory for downloads")

	rootCmd.AddCommand(searchCmd)
}
