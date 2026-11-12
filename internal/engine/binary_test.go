package engine

import (
	"io"
	"strings"
	"testing"
)

func TestProgressReaderReportsBytes(t *testing.T) {
	var msgs []string
	pr := &progressReader{
		r:        strings.NewReader(strings.Repeat("x", 4096)),
		total:    4096,
		onStatus: func(m string) { msgs = append(msgs, m) },
	}
	n, err := io.Copy(io.Discard, pr)
	if err != nil || n != 4096 {
		t.Fatalf("copy = %d, %v", n, err)
	}
	if len(msgs) == 0 || !strings.Contains(msgs[0], "downloading yt-dlp") || !strings.Contains(msgs[0], "/") {
		t.Errorf("expected a progress message with total, got %v", msgs)
	}
}
