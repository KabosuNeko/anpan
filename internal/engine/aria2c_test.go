package engine

import (
	"testing"
)

func TestParseUnitBytes(t *testing.T) {
	if got := ParseUnitBytes("0B"); got != 0 {
		t.Errorf("ParseUnitBytes('0B') = %v, want 0", got)
	}
	if got := ParseUnitBytes("500B"); got != 500 {
		t.Errorf("ParseUnitBytes('500B') = %v, want 500", got)
	}
	if got := ParseUnitBytes("10KiB"); got != 10240 {
		t.Errorf("ParseUnitBytes('10KiB') = %v, want 10240", got)
	}
	if got := ParseUnitBytes("1.5MiB"); got != 1572864 {
		t.Errorf("ParseUnitBytes('1.5MiB') = %v, want 1572864", got)
	}
	if got := ParseUnitBytes("1.4GiB"); got != 1503238554 {
		t.Errorf("ParseUnitBytes('1.4GiB') = %v, want 1503238554", got)
	}
}

func TestParseAriaEta(t *testing.T) {
	if got := ParseAriaEta("2s"); got == nil || *got != 2 {
		t.Errorf("ParseAriaEta('2s') = %v, want 2", got)
	}
	if got := ParseAriaEta("5m"); got == nil || *got != 300 {
		t.Errorf("ParseAriaEta('5m') = %v, want 300", got)
	}
	if got := ParseAriaEta("2m30s"); got == nil || *got != 150 {
		t.Errorf("ParseAriaEta('2m30s') = %v, want 150", got)
	}
	if got := ParseAriaEta("1h20m"); got == nil || *got != 4800 {
		t.Errorf("ParseAriaEta('1h20m') = %v, want 4800", got)
	}
}

func TestParseAriaProgressLine(t *testing.T) {
	line1 := "[#2089b0 1.2MiB/4.5MiB(26%) CN:16 DL:1.1MiB ETA:2s]"
	res1 := ParseAriaProgressLine(line1)
	if res1 == nil {
		t.Fatalf("ParseAriaProgressLine returned nil for %s", line1)
	}
	if res1.DownloadedBytes != 1258291 || res1.TotalBytes == nil || *res1.TotalBytes != 4718592 || res1.Connections != 16 || res1.Speed != 1153434 || res1.ETA == nil || *res1.ETA != 2 {
		t.Errorf("Mismatch for line1: %+v", res1)
	}

	line2 := "[#a77dff 12MiB/1.4GiB(1%) CN:5 SD:2 DL:1.2MiB UL:32KiB ETA:5m]"
	res2 := ParseAriaProgressLine(line2)
	if res2 == nil {
		t.Fatalf("ParseAriaProgressLine returned nil for %s", line2)
	}
	if res2.DownloadedBytes != 12582912 || res2.TotalBytes == nil || *res2.TotalBytes != 1503238554 || res2.Connections != 5 || res2.Seeders == nil || *res2.Seeders != 2 || res2.Speed != 1258291 || res2.ETA == nil || *res2.ETA != 300 {
		t.Errorf("Mismatch for line2: %+v", res2)
	}

	line3 := "[#a77dff 0B/1.4GiB(0%) CN:5 SD:0 DL:0B]"
	res3 := ParseAriaProgressLine(line3)
	if res3 == nil {
		t.Fatalf("ParseAriaProgressLine returned nil for %s", line3)
	}
	if res3.DownloadedBytes != 0 || res3.TotalBytes == nil || *res3.TotalBytes != 1503238554 || res3.Connections != 5 || res3.Seeders == nil || *res3.Seeders != 0 || res3.Speed != 0 {
		t.Errorf("Mismatch for line3: %+v", res3)
	}

	line4 := "[#13d9df 0B/0B CN:0 SD:0 DL:0B]"
	res4 := ParseAriaProgressLine(line4)
	if res4 == nil {
		t.Fatalf("ParseAriaProgressLine returned nil for %s", line4)
	}
	if res4.DownloadedBytes != 0 || res4.TotalBytes != nil || res4.Connections != 0 || res4.Seeders == nil || *res4.Seeders != 0 || res4.Speed != 0 {
		t.Errorf("Mismatch for line4: %+v", res4)
	}
}
