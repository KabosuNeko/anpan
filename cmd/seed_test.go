package cmd

import (
	"testing"
)

func TestSeedCmdFlags(t *testing.T) {
	if seedCmd.Use != "seed <file|folder>" {
		t.Errorf("expected seed command use, got %s", seedCmd.Use)
	}

	noSeedFlag := seedCmd.Flag("no-seed")
	if noSeedFlag == nil || noSeedFlag.DefValue != "false" {
		t.Errorf("expected no-seed flag with default false")
	}

	outFlag := seedCmd.Flag("out")
	if outFlag == nil {
		t.Errorf("expected out flag")
	}
}
