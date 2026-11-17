package node

import (
	"strings"
	"testing"
)

func TestNpmCIFallbackNote(t *testing.T) {
	note := npmCIFallbackNote()
	if strings.Contains(note, "\n") {
		t.Errorf("want a single-line message, got %q", note)
	}
	for _, want := range []string{"npm ci failed", "npm install instead", "lockfile"} {
		if !strings.Contains(note, want) {
			t.Errorf("note should contain %q, got %q", want, note)
		}
	}
}
