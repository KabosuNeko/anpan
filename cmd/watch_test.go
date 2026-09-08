package cmd

import (
	"testing"
)

func TestWatchCmdFlags(t *testing.T) {
	if watchCmd.Use != "watch [folder]" {
		t.Errorf("expected watch command use, got %s", watchCmd.Use)
	}

	connsFlag := watchCmd.Flag("connections")
	if connsFlag == nil || connsFlag.DefValue != "16" {
		t.Errorf("expected connections flag with default 16")
	}

	intervalFlag := watchCmd.Flag("interval")
	if intervalFlag == nil || intervalFlag.DefValue != "3" {
		t.Errorf("expected interval flag with default 3")
	}
}
