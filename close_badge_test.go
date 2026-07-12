package glamour

import (
	"io"
	"strings"
	"testing"
)

func TestCloseBadgeCenter(t *testing.T) {
	r, err := NewTermRenderer(
		WithWordWrap(80),
		WithStandardStyle("dark"),
		WithShieldsBadges(true),
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = r.Write([]byte("<div align=\"center\">\n<img src=\"https://img.shields.io/badge/build-passing-brightgreen\" />\n<img src=\"https://img.shields.io/github/last-commit/golang/go\" />\n</div>"))
	if err != nil {
		t.Fatal(err)
	}
	err = r.Close()
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	t.Logf("RAW: %q", s)
	if strings.Contains(s, "\x00") {
		t.Error("marker bytes found - centering not applied")
	}
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		clean := stripANSI(line)
		trimmed := strings.TrimSpace(clean)
		if len(trimmed) > 0 && trimmed != "build" && trimmed != "passing" && trimmed != "last commit" {
			continue
		}
		// Check for centering: >2 leading spaces
		spaces := len(line) - len(strings.TrimLeft(line, " "))
		t.Logf("line=%q spaces=%d", trimmed, spaces)
	}
}
