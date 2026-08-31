package system

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const maxHistoryEntries = 50

func historyPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "anpan", "history.json")
}

func LoadHistory() []string {
	data, err := os.ReadFile(historyPath())
	if err != nil {
		return []string{}
	}
	var res []string
	if err := json.Unmarshal(data, &res); err == nil {
		return res
	}
	return []string{}
}

func AddToHistory(url string) []string {
	history := LoadHistory()
	var updated []string
	updated = append(updated, url)
	for _, item := range history {
		if item != url {
			updated = append(updated, item)
		}
	}
	if len(updated) > maxHistoryEntries {
		updated = updated[:maxHistoryEntries]
	}

	p := historyPath()
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	if data, err := json.MarshalIndent(updated, "", "  "); err == nil {
		_ = os.WriteFile(p, append(data, '\n'), 0o644)
	}
	return updated
}
