package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/tui"
	"github.com/spf13/cobra"
)

var (
	Version   = "0.5.0"
	outputDir string
	inputFile string

	rootCmd = &cobra.Command{
		Use:   "anpan [url|magnet|file...]",
		Short: "anpan — feed a link, bake a file.",
		Long: `anpan — feed a link, bake a file.
(youtube · x · instagram · soundcloud · torrent · pixiv · and more)

Usage:
  anpan [url|magnet|file...] [options]
  anpan -i urls.txt

Examples:
  anpan https://youtu.be/dQw4w9WgXcQ
  anpan -o ~/Videos https://youtu.be/dQw4w9WgXcQ
  anpan -i list.txt
  anpan url1 url2 url3
  anpan                 (prompts for input)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var targets []string

			if inputFile != "" {
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("could not read input file: %w", err)
				}
				for _, line := range strings.Split(string(data), "\n") {
					trimmed := strings.TrimSpace(line)
					if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
						targets = append(targets, trimmed)
					}
				}
			}

			for _, arg := range args {
				trimmed := strings.TrimSpace(arg)
				if trimmed != "" {
					targets = append(targets, trimmed)
				}
			}

			if len(targets) <= 1 {
				initialURL := ""
				if len(targets) == 1 {
					initialURL = targets[0]
				}

				model := tui.NewModel(Version, initialURL, outputDir)
				p := tea.NewProgram(model)
				model.SetProgram(p)

				finalModel, err := p.Run()
				if err != nil {
					return err
				}

				if m, ok := finalModel.(tui.Model); ok && m.FinalPath != "" {
					fmt.Printf("done → %s\n", m.FinalPath)
				}
			} else {
				// Batch mode
				total := len(targets)
				fmt.Printf("anpan — batch queue (%d items)\n", total)
				for i, target := range targets {
					fmt.Printf("\n[%d/%d] → %s\n", i+1, total, target)
					model := tui.NewModel(Version, target, outputDir)
					p := tea.NewProgram(model)
					model.SetProgram(p)

					finalModel, err := p.Run()
					if err != nil {
						fmt.Printf("✗ failed: %v\n", err)
						continue
					}
					if m, ok := finalModel.(tui.Model); ok && m.FinalPath != "" {
						fmt.Printf("✓ done → %s\n", m.FinalPath)
					}
				}
				fmt.Printf("\n✓ batch queue completed (%d items)\n", total)
			}

			check := system.CheckUpdate(context.Background(), Version, nil)
			if check != nil && check.UpdateAvailable {
				fmt.Printf("\033[33m✦ update available:\033[0m %s → \033[32mv%s\033[0m (run: \033[1manpan update\033[0m)\n", Version, check.LatestVersion)
			}

			return nil
		},
	}
)

func init() {
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "", "specify download output directory")
	rootCmd.Flags().StringVar(&outputDir, "out-dir", "", "specify download output directory")
	rootCmd.Flags().StringVarP(&inputFile, "input", "i", "", "batch download from a text file (one URL per line)")
	rootCmd.Flags().StringVarP(&inputFile, "file", "f", "", "batch download from a text file (alias for -i)")
	rootCmd.Version = Version
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
