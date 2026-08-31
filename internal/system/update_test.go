package system

import (
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	// Major bump
	if !IsNewerVersion("1.0.0", "0.9.9") {
		t.Errorf("expected 1.0.0 > 0.9.9")
	}
	if IsNewerVersion("0.9.9", "1.0.0") {
		t.Errorf("expected 0.9.9 not > 1.0.0")
	}

	// Minor bump
	if !IsNewerVersion("0.2.0", "0.1.0") {
		t.Errorf("expected 0.2.0 > 0.1.0")
	}
	if IsNewerVersion("0.1.0", "0.2.0") {
		t.Errorf("expected 0.1.0 not > 0.2.0")
	}

	// Patch bump
	if !IsNewerVersion("0.1.1", "0.1.0") {
		t.Errorf("expected 0.1.1 > 0.1.0")
	}
	if IsNewerVersion("0.1.0", "0.1.1") {
		t.Errorf("expected 0.1.0 not > 0.1.1")
	}

	// Equal versions
	if IsNewerVersion("0.1.0", "0.1.0") {
		t.Errorf("expected equal versions to return false")
	}
	if IsNewerVersion("1.2.3", "1.2.3") {
		t.Errorf("expected equal versions to return false")
	}

	// Handles 'v' prefix
	if !IsNewerVersion("v0.2.0", "0.1.0") {
		t.Errorf("expected v0.2.0 > 0.1.0")
	}
	if !IsNewerVersion("0.2.0", "v0.1.0") {
		t.Errorf("expected 0.2.0 > v0.1.0")
	}
	if IsNewerVersion("v0.1.0", "v0.1.0") {
		t.Errorf("expected equal versions to return false")
	}

	// Two-digit parts
	if !IsNewerVersion("0.10.0", "0.9.0") {
		t.Errorf("expected 0.10.0 > 0.9.0")
	}
	if !IsNewerVersion("0.1.10", "0.1.9") {
		t.Errorf("expected 0.1.10 > 0.1.9")
	}
}

func TestParseSemver(t *testing.T) {
	maj, min, patch := parseSemver("v1.2.3")
	if maj != 1 || min != 2 || patch != 3 {
		t.Errorf("expected 1.2.3, got %d.%d.%d", maj, min, patch)
	}

	maj, min, patch = parseSemver("0.3.0")
	if maj != 0 || min != 3 || patch != 0 {
		t.Errorf("expected 0.3.0, got %d.%d.%d", maj, min, patch)
	}
}
