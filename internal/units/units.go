package units

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/mattn/go-runewidth"
)

func FormatBytes(bytes float64) string {
	if math.IsNaN(bytes) || math.IsInf(bytes, 0) || bytes <= 0 {
		return ""
	}
	tiers := []string{"B", "KB", "MB", "GB"}
	val := bytes
	tier := 0
	for val >= 1024 && tier < len(tiers)-1 {
		val /= 1024
		tier++
	}
	var formatted string
	if val >= 10 || tier == 0 || math.Floor(val) == val {
		formatted = fmt.Sprintf("%.0f", math.Round(val))
	} else {
		formatted = fmt.Sprintf("%.1f", val)
	}
	return fmt.Sprintf("%s %s", formatted, tiers[tier])
}

func FormatDuration(totalSeconds float64) string {
	if math.IsNaN(totalSeconds) || math.IsInf(totalSeconds, 0) || totalSeconds <= 0 {
		return ""
	}
	s := int64(math.Round(totalSeconds))
	hrs := s / 3600
	mins := (s % 3600) / 60
	secs := s % 60
	if hrs > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hrs, mins, secs)
	}
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func FormatSpeed(bytesPerSecond float64) string {
	if math.IsNaN(bytesPerSecond) || math.IsInf(bytesPerSecond, 0) || bytesPerSecond <= 0 {
		return ""
	}
	return fmt.Sprintf("%s/s", FormatBytes(bytesPerSecond))
}

func FormatEta(seconds float64) string {
	if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 {
		return ""
	}
	return FormatDuration(seconds)
}

func Truncate(text string, max int) string {
	if runewidth.StringWidth(text) <= max {
		return text
	}
	target := max - 1
	if target < 1 {
		target = 1
	}
	var b strings.Builder
	curWidth := 0
	for _, r := range text {
		w := runewidth.RuneWidth(r)
		if curWidth+w > target {
			break
		}
		b.WriteRune(r)
		curWidth += w
	}
	b.WriteString("…")
	return b.String()
}

func ShortenPath(filePath string, homeDir string, max ...int) string {
	maxLen := 60
	if len(max) > 0 && max[0] > 0 {
		maxLen = max[0]
	}
	normFile := filepath.Clean(filePath)
	normHome := filepath.Clean(homeDir)

	isUnderHome := false
	if runtime.GOOS == "windows" {
		lowerFile := strings.ToLower(normFile)
		lowerHome := strings.ToLower(normHome)
		isUnderHome = lowerFile == lowerHome ||
			strings.HasPrefix(lowerFile, lowerHome+string(filepath.Separator)) ||
			strings.HasPrefix(lowerFile, lowerHome+"/")
	} else {
		isUnderHome = normFile == normHome || strings.HasPrefix(normFile, normHome+string(filepath.Separator))
	}

	pretty := normFile
	if isUnderHome {
		pretty = "~" + normFile[len(normHome):]
	}

	if len(pretty) <= maxLen {
		return pretty
	}
	ext := filepath.Ext(pretty)
	trimLen := maxLen - len(ext) - 1
	if trimLen < 1 {
		trimLen = 1
	}
	return fmt.Sprintf("%s…%s", pretty[:trimLen], ext)
}

func ResolveUserPath(raw string, customHome ...string) string {
	home := ""
	if len(customHome) > 0 && customHome[0] != "" {
		home = customHome[0]
	} else {
		home, _ = os.UserHomeDir()
	}
	cleaned := strings.TrimSpace(raw)
	if strings.HasPrefix(cleaned, "~") {
		rest := strings.TrimLeft(cleaned[1:], "/\\")
		cleaned = filepath.Join(home, rest)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return cleaned
	}
	return abs
}

var whitespaceRegex = regexp.MustCompile(`\s+`)

func WrapText(text string, width int) []string {
	var lines []string
	words := whitespaceRegex.Split(strings.TrimSpace(text), -1)
	current := ""
	for _, word := range words {
		if word == "" {
			continue
		}
		if current == "" {
			current = word
		} else if runewidth.StringWidth(current)+1+runewidth.StringWidth(word) <= width {
			current += " " + word
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}
