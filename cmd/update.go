package cmd

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/KabosuNeko/anpan/internal/system"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update anpan to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("✦ Checking for updates (current version: %s)...\n", Version)
		res := system.CheckUpdate(context.Background(), Version, &system.CheckUpdateOptions{Force: true})
		if res == nil {
			return fmt.Errorf("could not check latest version")
		}

		if !res.UpdateAvailable {
			fmt.Printf("✓ You are already on the latest version (%s)\n", Version)
			return nil
		}

		fmt.Printf("→ New version available: %s → v%s\n", Version, res.LatestVersion)

		execPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("cannot locate current binary: %w", err)
		}
		execPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return fmt.Errorf("cannot resolve symlink: %w", err)
		}

		// Platform asset
		osName := runtime.GOOS
		archName := runtime.GOARCH

		assetExt := ".tar.gz"
		if osName == "windows" {
			assetExt = ".zip"
		}
		assetName := fmt.Sprintf("anpan-%s-%s%s", osName, archName, assetExt)
		downloadURL := fmt.Sprintf("https://github.com/KabosuNeko/anpan/releases/download/v%s/%s", res.LatestVersion, assetName)

		fmt.Printf("→ Downloading %s ...\n", downloadURL)
		req, err := http.NewRequestWithContext(context.Background(), "GET", downloadURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "anpan/"+Version)
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			return fmt.Errorf("download failed (HTTP %d). Please install manually via install script.", resp.StatusCode)
		}
		defer resp.Body.Close()

		tmpArchive, err := os.CreateTemp("", "anpan-update-*"+assetExt)
		if err != nil {
			return err
		}
		defer os.Remove(tmpArchive.Name())

		if _, err := io.Copy(tmpArchive, resp.Body); err != nil {
			tmpArchive.Close()
			return err
		}
		tmpArchive.Close()

		tmpExtract, err := os.MkdirTemp("", "anpan-extract-*")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpExtract)

		if strings.HasSuffix(assetExt, ".tar.gz") {
			tarCmd := exec.Command("tar", "-xzf", tmpArchive.Name(), "-C", tmpExtract)
			if err := tarCmd.Run(); err != nil {
				return fmt.Errorf("tar extract failed: %w", err)
			}
		} else if strings.HasSuffix(assetExt, ".zip") {
			if runtime.GOOS == "windows" {
				psCmd := exec.Command("powershell", "-Command", fmt.Sprintf("Expand-Archive -Path '%s' -DestinationPath '%s' -Force", tmpArchive.Name(), tmpExtract))
				if err := psCmd.Run(); err != nil {
					return fmt.Errorf("powershell unzip failed: %w", err)
				}
			} else {
				unzipCmd := exec.Command("unzip", "-o", tmpArchive.Name(), "-d", tmpExtract)
				if err := unzipCmd.Run(); err != nil {
					return fmt.Errorf("unzip failed: %w", err)
				}
			}
		}

		extractedBin := filepath.Join(tmpExtract, "anpan")
		if runtime.GOOS == "windows" {
			extractedBin += ".exe"
		}

		if _, err := os.Stat(extractedBin); err != nil {
			return fmt.Errorf("binary not found in archive: %w", err)
		}

		// Replace current executable
		oldPath := execPath + ".old"
		_ = os.Remove(oldPath)
		_ = os.Rename(execPath, oldPath)

		input, err := os.ReadFile(extractedBin)
		if err != nil {
			_ = os.Rename(oldPath, execPath)
			return err
		}

		if err := os.WriteFile(execPath, input, 0o755); err != nil {
			_ = os.Rename(oldPath, execPath)
			return fmt.Errorf("failed to write binary (try running with sudo?): %w", err)
		}

		_ = os.Remove(oldPath)
		fmt.Printf("✓ Successfully updated anpan to v%s!\n", res.LatestVersion)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(updateCmd)
}
