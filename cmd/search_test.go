package cmd

import (
	"testing"
)

func TestSearchCmdFlags(t *testing.T) {
	if searchCmd.Use != "search <query>" {
		t.Errorf("expected search command use, got %s", searchCmd.Use)
	}

	catFlag := searchCmd.Flag("category")
	if catFlag == nil || catFlag.DefValue != "all" {
		t.Errorf("expected category flag with default 'all'")
	}

	jsonFlag := searchCmd.Flag("json")
	if jsonFlag == nil || jsonFlag.DefValue != "false" {
		t.Errorf("expected json flag with default false")
	}
}
