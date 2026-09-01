package engine

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func FindFfmpeg() string {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return filepath.Dir(p)
	}

	commonPaths := []string{
		"/usr/bin/ffmpeg",
		"/usr/local/bin/ffmpeg",
		"/usr/sbin/ffmpeg",
		"/opt/homebrew/bin/ffmpeg",
	}
	for _, cp := range commonPaths {
		if BinaryResponds(cp, "-version") {
			return filepath.Dir(cp)
		}
	}

	binDir := AnpanBinDir()
	ffmpegName := "ffmpeg"
	ffprobeName := "ffprobe"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
		ffprobeName = "ffprobe.exe"
	}

	ffmpegBin := filepath.Join(binDir, ffmpegName)
	ffprobeBin := filepath.Join(binDir, ffprobeName)

	if BinaryResponds(ffmpegBin, "-version") && BinaryResponds(ffprobeBin, "-version") {
		return binDir
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		winPaths := []string{
			filepath.Join(home, "AppData", "Local", "Microsoft", "WinGet", "Links"),
			filepath.Join(home, "scoop", "shims"),
			`C:\ProgramData\chocolatey\bin`,
		}
		for _, wp := range winPaths {
			if BinaryResponds(filepath.Join(wp, "ffmpeg.exe"), "-version") {
				return wp
			}
		}
	}

	return ""
}
