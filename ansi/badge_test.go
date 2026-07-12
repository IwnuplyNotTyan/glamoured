package ansi

import (
	"testing"
)

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
	}
	for _, tt := range tests {
		got := hexToANSI(tt.hex)
		if got != tt.want {
			t.Errorf("hexToANSI(%q) = %d, want %d", tt.hex, got, tt.want)
		}
	}
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
