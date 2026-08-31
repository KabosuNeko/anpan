package units

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{0, ""},
		{-1, ""},
		{500, "500 B"},
		{1024, "1 KB"},
		{1536, "1.5 KB"},
		{10240, "10 KB"},
		{1048576, "1 MB"},
		{1073741824, "1 GB"},
	}

	for _, tc := range cases {
		got := FormatBytes(tc.input)
		if got != tc.want {
			t.Errorf("FormatBytes(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		input float64
		want  string
	}{
		{0, ""},
		{-5, ""},
		{5, "0:05"},
		{65, "1:05"},
		{3661, "1:01:01"},
	}

	for _, tc := range cases {
		got := FormatDuration(tc.input)
		if got != tc.want {
			t.Errorf("FormatDuration(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFormatSpeedAndEta(t *testing.T) {
	if got := FormatSpeed(1048576); got != "1 MB/s" {
		t.Errorf("FormatSpeed(1048576) = %q, want 1 MB/s", got)
	}
	if got := FormatSpeed(0); got != "" {
		t.Errorf("FormatSpeed(0) = %q, want empty", got)
	}
	if got := FormatEta(90); got != "1:30" {
		t.Errorf("FormatEta(90) = %q, want 1:30", got)
	}
	if got := FormatEta(0); got != "" {
		t.Errorf("FormatEta(0) = %q, want empty", got)
	}
}

func TestTruncate(t *testing.T) {
	if got := Truncate("hello", 10); got != "hello" {
		t.Errorf("Truncate(hello, 10) = %q, want hello", got)
	}
	if got := Truncate("hello world", 8); got != "hello w…" {
		t.Errorf("Truncate(hello world, 8) = %q, want 'hello w…'", got)
	}
	if got := Truncate("澤野弘之", 6); got != "澤野…" {
		t.Errorf("Truncate(澤野弘之, 6) = %q, want '澤野…'", got)
	}
}

func TestShortenPath(t *testing.T) {
	home := "/home/user"
	file := "/home/user/Downloads/file.mp4"
	expected := "~/Downloads/file.mp4"
	if runtime.GOOS == "windows" {
		home = `C:\Users\test`
		file = `C:\Users\test\Downloads\file.mp4`
		expected = `~\Downloads\file.mp4`
	}

	got := ShortenPath(file, home)
	if got != expected {
		t.Errorf("ShortenPath(%q, %q) = %q, want %q", file, home, got, expected)
	}

	other := "/other/path/file.mp4"
	if runtime.GOOS == "windows" {
		other = `D:\other\path\file.mp4`
	}
	gotOther := ShortenPath(other, home)
	if gotOther != filepath.Clean(other) {
		t.Errorf("ShortenPath(%q, %q) = %q, want %q", other, home, gotOther, filepath.Clean(other))
	}
}

func TestResolveUserPath(t *testing.T) {
	fakeHome := "/mock/home/user"
	if runtime.GOOS == "windows" {
		fakeHome = `C:\mock\home\user`
	}
	wantMusic, _ := filepath.Abs(filepath.Join(fakeHome, "Music"))
	wantHome, _ := filepath.Abs(fakeHome)

	if got := ResolveUserPath("~/Music", fakeHome); got != wantMusic {
		t.Errorf("ResolveUserPath(~/Music) = %q, want %q", got, wantMusic)
	}
	if got := ResolveUserPath("~", fakeHome); got != wantHome {
		t.Errorf("ResolveUserPath(~) = %q, want %q", got, wantHome)
	}
}

func TestWrapText(t *testing.T) {
	lines := WrapText("a b c d e", 5)
	if len(lines) != 2 || lines[0] != "a b c" || lines[1] != "d e" {
		t.Errorf("WrapText('a b c d e', 5) = %v", lines)
	}
	short := WrapText("short", 20)
	if len(short) != 1 || short[0] != "short" {
		t.Errorf("WrapText('short', 20) = %v", short)
	}
	spaced := WrapText("  spaced  out  ", 20)
	if len(spaced) != 1 || spaced[0] != "spaced out" {
		t.Errorf("WrapText('  spaced  out  ', 20) = %v", spaced)
	}
}
