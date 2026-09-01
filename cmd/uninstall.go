package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	purgeUninstall bool
	yesUninstall   bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall anpan and optionally remove all configs and cache",
	RunE: func(cmd *cobra.Command, args []string) error {
		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot locate current binary: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("cannot resolve symlink: %w", err)
		}

		// Detect if installed via package manager (e.g. Arch Linux AUR / pacman)
		if runtime.GOOS == "linux" && (strings.HasPrefix(execPath, "/usr/bin/") || strings.HasPrefix(execPath, "/usr/local/bin/")) {
			if _, pErr := exec.LookPath("pacman"); pErr == nil {
				checkCmd := exec.Command("pacman", "-Qo", execPath)
				if err := checkCmd.Run(); err == nil {
					fmt.Println("✦ anpan was installed via package manager (Arch Linux / AUR).")
					fmt.Println("→ Please uninstall using your package manager or AUR helper:")
					fmt.Println("   yay -R anpan-git    (or paru -R anpan-git / sudo pacman -R anpan-git)")
					return nil
				}
			}
		}

		if !yesUninstall {
			fmt.Printf("Are you sure you want to uninstall anpan at %s? [y/N]: ", execPath)
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				input := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if input != "y" && input != "yes" {
					fmt.Println("Uninstall cancelled.")
					return nil
				}
			} else {
				fmt.Println("Uninstall cancelled.")
				return nil
			}
		}

		// Remove binary
		if err := os.Remove(execPath); err != nil {
			return fmt.Errorf("failed to remove binary %s (try running with sudo?): %w", execPath, err)
		}
		fmt.Printf("✓ Removed binary: %s\n", execPath)

		home, _ := os.UserHomeDir()
		anpanBinDir := filepath.Join(home, ".anpan")
		configDir := filepath.Join(home, ".config", "anpan")

		if purgeUninstall {
			if _, err := os.Stat(anpanBinDir); err == nil {
				_ = os.RemoveAll(anpanBinDir)
				fmt.Printf("✓ Removed cache dir: %s\n", anpanBinDir)
			}
			if _, err := os.Stat(configDir); err == nil {
				_ = os.RemoveAll(configDir)
				fmt.Printf("✓ Removed config dir: %s\n", configDir)
			}
		} else {
			if _, err := os.Stat(configDir); err == nil {
				fmt.Printf("✦ Preserved configuration in: %s\n", configDir)
				fmt.Println("  (To remove configuration as well, run: anpan uninstall --purge)")
			}
		}

		fmt.Println("✓ anpan has been successfully uninstalled.")
		return nil
	},
}

func init() {
	uninstallCmd.Flags().BoolVarP(&purgeUninstall, "purge", "p", false, "also remove configuration and cached binaries (~/.config/anpan, ~/.anpan)")
	uninstallCmd.Flags().BoolVarP(&yesUninstall, "yes", "y", false, "skip confirmation prompt")
	rootCmd.AddCommand(uninstallCmd)
}
