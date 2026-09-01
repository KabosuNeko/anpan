package engine

import (
	"context"
	"testing"
)

func TestIsArchiveOrgURL(t *testing.T) {
	if !IsArchiveOrgURL("https://archive.org/details/msdos_Oregon_Trail_The_1990") {
		t.Errorf("expected true for archive.org details URL")
	}
	if IsArchiveOrgURL("https://youtube.com/watch?v=123") {
		t.Errorf("expected false for youtube URL")
	}
}

func TestProbeArchiveOrgLive(t *testing.T) {
	ctx := context.Background()
	post, err := ProbeArchiveOrg(ctx, "https://archive.org/details/msdos_Oregon_Trail_The_1990")
	if err != nil {
		t.Logf("Archive.org live probe skipped: %v", err)
		return
	}
	if len(post.Files) == 0 {
		t.Errorf("expected files in archive.org item")
	}
}
