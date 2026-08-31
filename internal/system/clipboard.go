package system

import (
	"strings"

	"github.com/atotto/clipboard"
)

func ReadClipboard() string {
	text, err := clipboard.ReadAll()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(text)
}
