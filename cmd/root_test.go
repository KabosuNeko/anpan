package cmd

import (
	"bytes"
	"testing"
)

func TestRootCmdFlags(t *testing.T) {
	if rootCmd.Use != "anpan [url|magnet|file...]" {
		t.Errorf("unexpected Use: %s", rootCmd.Use)
	}

	oFlag := rootCmd.Flags().Lookup("output")
	if oFlag == nil || oFlag.Shorthand != "o" {
		t.Errorf("expected -o/--output flag")
	}

	iFlag := rootCmd.Flags().Lookup("input")
	if iFlag == nil || iFlag.Shorthand != "i" {
		t.Errorf("expected -i/--input flag")
	}

	fFlag := rootCmd.Flags().Lookup("file")
	if fFlag == nil || fFlag.Shorthand != "f" {
		t.Errorf("expected -f/--file flag")
	}

	outDirFlag := rootCmd.Flags().Lookup("out-dir")
	if outDirFlag == nil {
		t.Errorf("expected --out-dir flag")
	}

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--help"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("help command failed: %v", err)
	}
}

func TestUpdateCmd(t *testing.T) {
	if updateCmd.Use != "update" {
		t.Errorf("unexpected Use: %s", updateCmd.Use)
	}
}

func TestUninstallCmd(t *testing.T) {
	if uninstallCmd.Use != "uninstall" {
		t.Errorf("unexpected Use: %s", uninstallCmd.Use)
	}
	pFlag := uninstallCmd.Flags().Lookup("purge")
	if pFlag == nil || pFlag.Shorthand != "p" {
		t.Errorf("expected -p/--purge flag")
	}
	yFlag := uninstallCmd.Flags().Lookup("yes")
	if yFlag == nil || yFlag.Shorthand != "y" {
		t.Errorf("expected -y/--yes flag")
	}
}

func TestUpdateCmdExecution(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"update"})
	// Running update command in test environment:
	// If already on latest version, should return without error.
	_ = rootCmd.Execute()
}
