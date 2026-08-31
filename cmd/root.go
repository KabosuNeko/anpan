package cmd

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/KabosuNeko/anpan/internal/tui"
	"github.com/spf13/cobra"
)

var (
	Version   = "0.4.0"
	outputDir string

	rootCmd = &cobra.Command{
		Use:   "anpan [url|magnet|file]",
		Short: "anpan — feed a link, bake a file.",
		Long: `anpan — feed a link, bake a file.
(youtube · x · instagram · soundcloud · torrent · and more)

Usage:
  anpan [url|magnet|file] [options]

Examples:
  anpan https://youtu.be/dQw4w9WgXcQ
  anpan https://youtu.be/dQw4w9WgXcQ -o ~/Videos
  anpan "magnet:?xt=urn:btih:..."
  anpan https://example.com/nixos-minimal.iso
  anpan                 (prompts for input)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			initialURL := ""
			if len(args) > 0 {
				initialURL = args[0]
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
	rootCmd.Version = Version
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
