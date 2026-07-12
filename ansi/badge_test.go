package ansi

import (
	"strings"
	"testing"
)

func TestIsShieldsURL(t *testing.T) {
	tests := []struct {
		url  string
		want bool
	}{
		{"https://img.shields.io/badge/Go-1.21-blue", true},
		{"https://img.shields.io/github/license/owner/repo", true},
		{"https://example.com/image.png", false},
		{"not-a-url", false},
	}
	for _, tt := range tests {
		got := isShieldsURL(tt.url)
		if got != tt.want {
			t.Errorf("isShieldsURL(%q) = %v, want %v", tt.url, got, tt.want)
		}
	}
}

func TestParseShieldsColor(t *testing.T) {
	tests := []struct {
		color string
		want  int
	}{
		{"blue", 32},
		{"#007ec6", 31},
		{"007ec6", 31},
		{"ff0000", 196},
		{"f00", 196},
		{"#f00", 196},
		{"unknown", 240},
	}
	for _, tt := range tests {
		got := parseShieldsColor(tt.color)
		if got != tt.want {
			t.Errorf("parseShieldsColor(%q) = %d, want %d", tt.color, got, tt.want)
		}
	}
}

func TestParseShieldsURL(t *testing.T) {
	tests := []struct {
		url       string
		wantLabel string
		wantMsg   string
		wantColor string
		wantLogo  string
		wantOK    bool
	}{
		{
			url:       "https://img.shields.io/badge/Go-1.21-blue",
			wantLabel: "Go",
			wantMsg:   "1.21",
			wantColor: "blue",
			wantOK:    true,
		},
		{
			url:       "https://img.shields.io/badge/License-MIT-brightgreen",
			wantLabel: "License",
			wantMsg:   "MIT",
			wantColor: "brightgreen",
			wantOK:    true,
		},
		{
			url:       "https://img.shields.io/badge/Go_Releases-1.21--beta-brightgreen",
			wantLabel: "Go Releases",
			wantMsg:   "1.21-beta",
			wantColor: "brightgreen",
			wantOK:    true,
		},
		{
			url:       "https://img.shields.io/badge/hello__world-foo-ff69b4",
			wantLabel: "hello_world",
			wantMsg:   "foo",
			wantColor: "ff69b4",
			wantOK:    true,
		},
		{
			url:       "https://img.shields.io/badge/Go-1.21-blue?logo=go&style=flat",
			wantLabel: "Go",
			wantMsg:   "1.21",
			wantColor: "blue",
			wantLogo:  "go",
			wantOK:    true,
		},
		{
			url:   "https://example.com/image.png",
			wantOK: false,
		},
		{
			url:   "https://img.shields.io/endpoint?url=...",
			wantOK: false,
		},
		{
			url:   "https://img.shields.io/badge/",
			wantOK: false,
		},
		{
			url:   "https://img.shields.io/badge/Go-1.21-007ec6?logo=go",
			wantLabel: "Go",
			wantMsg:   "1.21",
			wantColor: "007ec6",
			wantLogo:  "go",
			wantOK:    true,
		},
		{
			url:   "https://img.shields.io/badge/one--two-v3--4--5-blue",
			wantLabel: "one-two",
			wantMsg:   "v3-4-5",
			wantColor: "blue",
			wantOK:    true,
		},
		{
			url:   "https://img.shields.io/badge/Go-1.21.3-blue",
			wantLabel: "Go",
			wantMsg:   "1.21.3",
			wantColor: "blue",
			wantOK:    true,
		},
		{
			url:   "https://img.shields.io/badge/one-two-three-blue",
			wantOK: false,
		},
		{
			url:   "https://img.shields.io/badge/a-b",
			wantOK: false,
		},
	}
	for _, tt := range tests {
		label, msg, color, logo, ok := parseShieldsURL(tt.url)
		if ok != tt.wantOK {
			t.Errorf("parseShieldsURL(%q) ok = %v, want %v", tt.url, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if label != tt.wantLabel {
			t.Errorf("parseShieldsURL(%q) label = %q, want %q", tt.url, label, tt.wantLabel)
		}
		if msg != tt.wantMsg {
			t.Errorf("parseShieldsURL(%q) msg = %q, want %q", tt.url, msg, tt.wantMsg)
		}
		if color != tt.wantColor {
			t.Errorf("parseShieldsURL(%q) color = %q, want %q", tt.url, color, tt.wantColor)
		}
		if logo != tt.wantLogo {
			t.Errorf("parseShieldsURL(%q) logo = %q, want %q", tt.url, logo, tt.wantLogo)
		}
	}
}

func TestBadgeNamedColor(t *testing.T) {
	tests := []struct {
		name string
		want int
	}{
		{"brightgreen", 2},
		{"blue", 32},
		{"red", 196},
		{"unknown", 240},
	}
	for _, tt := range tests {
		got := badgeNamedColor(tt.name)
		if got != tt.want {
			t.Errorf("badgeNamedColor(%q) = %d, want %d", tt.name, got, tt.want)
		}
	}
}

func TestHexToANSI(t *testing.T) {
	tests := []struct {
		hex  string
		want int
	}{
		{"000000", 16},
		{"ffffff", 231},
		{"ff0000", 196},
		{"invalid", 240},
		{"ff", 240},
		{"#ff0000", 196},
	}
	for _, tt := range tests {
		got := hexToANSI(tt.hex)
		if got != tt.want {
			t.Errorf("hexToANSI(%q) = %d, want %d", tt.hex, got, tt.want)
		}
	}
}

func TestRenderBadge(t *testing.T) {
	t.Run("without icon", func(t *testing.T) {
		var buf strings.Builder
		renderBadge(&buf, "Go", "1.21", 32, "")
		out := buf.String()
		if !strings.Contains(out, "Go") || !strings.Contains(out, "1.21") {
			t.Errorf("badge output missing label/message: %q", out)
		}
		if !strings.Contains(out, "\x1b[") {
			t.Errorf("badge output missing ANSI escapes: %q", out)
		}
	})

	t.Run("with icon", func(t *testing.T) {
		var buf strings.Builder
		renderBadge(&buf, "Go", "1.21", 32, "\ue61b")
		out := buf.String()
		if !strings.Contains(out, "\ue61b") {
			t.Errorf("badge output missing icon: %q", out)
		}
	})

	t.Run("line break prefix", func(t *testing.T) {
		var buf strings.Builder
		renderBadge(&buf, "Go", "1.21", 32, "")
		out := buf.String()
		if out[0] != '\n' {
			t.Errorf("badge should start with newline, got: %q", out)
		}
	})
}

func TestTryRenderBadge(t *testing.T) {
	t.Run("renders shield badge", func(t *testing.T) {
		e := &ImageElement{
			URL: "https://img.shields.io/badge/Go-1.21-blue",
		}
		ctx := NewRenderContext(Options{
			ShieldsBadges: true,
			MosaicEnabled: false,
		})
		var buf strings.Builder
		result := e.tryRenderBadge(&buf, ctx)
		if !result {
			t.Fatal("tryRenderBadge returned false")
		}
		out := buf.String()
		if !strings.Contains(out, "Go") || !strings.Contains(out, "1.21") {
			t.Errorf("badge output missing content: %q", out)
		}
	})

	t.Run("skips non-shield URL", func(t *testing.T) {
		e := &ImageElement{
			URL: "https://example.com/image.png",
		}
		ctx := NewRenderContext(Options{
			ShieldsBadges: true,
		})
		var buf strings.Builder
		result := e.tryRenderBadge(&buf, ctx)
		if result {
			t.Fatal("tryRenderBadge should return false for non-shield URL")
		}
	})

	t.Run("skips when disabled", func(t *testing.T) {
		e := &ImageElement{
			URL: "https://img.shields.io/badge/Go-1.21-blue",
		}
		ctx := NewRenderContext(Options{
			ShieldsBadges: false,
		})
		var buf strings.Builder
		result := e.tryRenderBadge(&buf, ctx)
		if result {
			t.Fatal("tryRenderBadge should return false when disabled")
		}
	})

	t.Run("uses Nerd Font icon when enabled", func(t *testing.T) {
		e := &ImageElement{
			URL: "https://img.shields.io/badge/Go-1.21-blue?logo=go",
		}
		ctx := NewRenderContext(Options{
			ShieldsBadges: true,
			NerdFontIcons: true,
		})
		var buf strings.Builder
		result := e.tryRenderBadge(&buf, ctx)
		if !result {
			t.Fatal("tryRenderBadge returned false")
		}
		out := buf.String()
		if !strings.Contains(out, "\ue61b") {
			t.Errorf("expected Nerd Font icon in output: %q", out)
		}
	})

	t.Run("no icon when Nerd Font disabled", func(t *testing.T) {
		e := &ImageElement{
			URL: "https://img.shields.io/badge/Go-1.21-blue?logo=go",
		}
		ctx := NewRenderContext(Options{
			ShieldsBadges: true,
			NerdFontIcons: false,
		})
		var buf strings.Builder
		result := e.tryRenderBadge(&buf, ctx)
		if !result {
			t.Fatal("tryRenderBadge returned false")
		}
		out := buf.String()
		if strings.Contains(out, "\ue61b") {
			t.Errorf("expected no Nerd Font icon when disabled: %q", out)
		}
	})
}

func TestLogoNerdIcon(t *testing.T) {
	tests := []struct {
		logo string
		want string
	}{
		{"go", "\ue61b"},
		{"", ""},
		{"unknown", "\uf0a3"},
	}
	for _, tt := range tests {
		got := logoNerdIcon(tt.logo)
		if got != tt.want {
			t.Errorf("logoNerdIcon(%q) = %q, want %q", tt.logo, got, tt.want)
		}
	}
}
