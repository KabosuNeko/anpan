package engine

import "testing"

func skipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping live network test in short mode")
	}
}
