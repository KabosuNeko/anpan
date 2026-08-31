package engine

import (
	"path/filepath"
	"runtime"
)

func FindFfmpeg() string {
	if BinaryResponds("ffmpeg", "-version") && BinaryResponds("ffprobe", "-version") {
		return ""
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

	return ""
}
