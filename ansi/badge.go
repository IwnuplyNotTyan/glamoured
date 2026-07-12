package ansi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	shieldTitleRe  = regexp.MustCompile(`<title>([^:]+): ([^<]*)</title>`)
	shieldAriaRe   = regexp.MustCompile(`aria-label="([^:]+): ([^"]*)"`)
	shieldFillRe   = regexp.MustCompile(`<rect[^>]*fill="([^"]+)"`)
)

// isShieldsURL reports whether the URL points to shields.io.
func isShieldsURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	return err == nil && u.Host == "img.shields.io"
}

// fetchShieldsBadge fetches a shields.io SVG badge and extracts label, message, and color.
// Used for dynamic badges (e.g. /github/license/...) that can't be parsed from URL alone.
func fetchShieldsBadge(rawURL string) (label, message string, color int, ok bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", "", 0, false
	}
	req.Header.Set("Accept", "image/svg+xml")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", 0, false
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, false
	}
	// Extract label and message from <title>LABEL: MESSAGE</title> or aria-label
	m := shieldTitleRe.FindSubmatch(body)
	if m == nil {
		m = shieldAriaRe.FindSubmatch(body)
	}
	if m == nil {
		return "", "", 0, false
	}
	label = string(m[1])
	message = string(m[2])
	// Extract message background color from second <rect fill="COLOR">
	fills := shieldFillRe.FindAllSubmatch(body, -1)
	if len(fills) < 2 {
		return "", "", 0, false
	}
	colorStr := string(fills[1][1])
	return label, message, parseShieldsColor(colorStr), true
}

// parseShieldsColor converts a shields.io color string (named or hex) to ANSI 256.
func parseShieldsColor(s string) int {
	s = strings.TrimSpace(s)
	if c, ok := badgeNamedColors[strings.ToLower(s)]; ok {
		return c
	}
	s = strings.TrimPrefix(s, "#")
	if len(s) == 6 && isHex(s) {
		return hexToANSI(s)
	}
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
		if isHex(s) {
			return hexToANSI(s)
		}
	}
	return 240
}

// parseShieldsURL parses a shields.io static badge URL.
// Returns label, message, color, logo name, and whether parsing succeeded.
func parseShieldsURL(rawURL string) (label, message, color, logo string, ok bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", "", false
	}
	if u.Host != "img.shields.io" {
		return "", "", "", "", false
	}
	path := strings.TrimPrefix(u.Path, "/badge/")
	if path == u.Path || path == "" {
		return "", "", "", "", false
	}
	// Literal dashes in values are encoded as --.
	// After replacing -- with sentinel, ALL remaining dashes are field separators.
	path = strings.ReplaceAll(path, "--", "\x00")
	firstSep := strings.IndexByte(path, '-')
	if firstSep < 0 {
		return "", "", "", "", false
	}
	secondSep := strings.IndexByte(path[firstSep+1:], '-')
	if secondSep < 0 {
		return "", "", "", "", false
	}
	secondSep += firstSep + 1
	// Ensure no extra separators remain
	if strings.IndexByte(path[secondSep+1:], '-') >= 0 {
		return "", "", "", "", false
	}
	label = strings.ReplaceAll(path[:firstSep], "\x00", "-")
	message = strings.ReplaceAll(path[firstSep+1:secondSep], "\x00", "-")
	color = path[secondSep+1:]
	// Decode underscores in label and message
	label = decodeShieldsValue(label)
	message = decodeShieldsValue(message)
	// Extract logo from query params
	logo = u.Query().Get("logo")
	return label, message, color, logo, true
}

// decodeShieldsValue decodes a single badge component value:
//   __ -> literal underscore
//   _  -> space
func decodeShieldsValue(s string) string {
	s = strings.ReplaceAll(s, "__", "\x01")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "\x01", "_")
	return s
}

// Named shields.io colors mapped to ANSI 256-color codes.
// Derived from https://shields.io/badges
var badgeNamedColors = map[string]int{
	"brightgreen": 2,   // #44CC11
	"green":       106, // #97CA00
	"yellowgreen": 142, // #A4A61D
	"yellow":      214, // #DFB317
	"orange":      208, // #FE7D37
	"red":         196, // #E05D44
	"blue":        32,  // #007EC6
	"lightgrey":   250, // #9F9F9F
	"grey":        240, // #555555
	"blueviolet":  99,  // #800080
	"pink":        205, // #E04E8C
	"cyan":        39,  // #00BFFF
	"purple":      93,  // #8A2BE2
}

func badgeNamedColor(name string) int {
	if c, ok := badgeNamedColors[name]; ok {
		return c
	}
	return 240 // default dark grey
}

func hexToANSI(hex string) int {
	if len(hex) == 7 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 240
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return 240
		}
	}
	r := parseHexByte(hex[0:2])
	g := parseHexByte(hex[2:4])
	b := parseHexByte(hex[4:6])
	return closestANSI256(r, g, b)
}

func parseHexByte(s string) byte {
	b, _ := strconv.ParseUint(s, 16, 8)
	return byte(b)
}

// closestANSI256 returns the closest ANSI 256-color code to the given RGB.
func closestANSI256(r, g, b byte) int {
	// 6×6×6 color cube: 16 + 36*r + 6*g + b
	cr := int(r) * 5 / 255
	cg := int(g) * 5 / 255
	cb := int(b) * 5 / 255
	return 16 + 36*cr + 6*cg + cb
}

// logoNerdIcon maps shields.io logo names to Nerd Font Unicode codepoints.
func logoNerdIcon(logo string) string {
	if logo == "" {
		return ""
	}
	if icon, ok := badgeLogoIcons[strings.ToLower(logo)]; ok {
		return icon
	}
	return "\uf0a3" // generic certificate icon
}

// shieldFallbackLabel extracts a meaningful label and message from a shields.io URL
// when SVG fetching/parsing fails. Handles paths like:
//
//	/github/stars/USER/REPO          → label="stars", message="USER/REPO"
//	/github/license/USER/REPO        → label="license", message="USER/REPO"
//	/github/actions/workflow/...     → label="build"
//	/github/v/release/...            → label="release"
func shieldFallbackLabel(rawURL string) (label, message string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	// Check for specific known patterns
	for i, p := range parts {
		switch p {
		case "actions":
			if i+1 < len(parts) && parts[i+1] == "workflow" {
				return "build", ""
			}
		case "v":
			if i+1 < len(parts) && parts[i+1] == "release" {
				return "release", ""
			}
		}
	}
	// Generic: skip common prefixes, use first meaningful segment
	for _, p := range parts {
		switch p {
		case "github", "img", "shields", "io", "badge", "workflow", "status":
			continue
		}
		label = p
		label = strings.ReplaceAll(label, "-", " ")
		label = strings.ReplaceAll(label, "_", " ")
		return label, ""
	}
	return "", ""
}

// renderBadge writes a shields.io-style badge to w.
// Format: \n[grey bg white fg] icon? LABEL [color bg white fg] MESSAGE [reset]
func renderBadge(w io.Writer, label, message string, color int, icon string) {
	labelBg := 240 // dark grey
	fg := 97       // bright white
	iconPart := icon
	if iconPart != "" {
		iconPart += " "
	}
	_, _ = fmt.Fprintf(w, "\n\033[48;5;%d;38;5;%dm %s%s \033[0m\033[48;5;%d;38;5;%dm %s \033[0m",
		labelBg, fg, iconPart, label, color, fg, message)
}

// badgeLogoIcons maps logo names to Nerd Font icon strings.
// Uses Font Awesome and Devicons codepoints from the Nerd Font PUA range.
var badgeLogoIcons = map[string]string{
	"go":         "\ue61b",
	"golang":     "\ue61b",
	"rust":       "\ue7a8",
	"python":     "\ue73c",
	"node":       "\ue718",
	"nodejs":     "\ue718",
	"javascript": "\ue74e",
	"js":         "\ue74e",
	"typescript": "\ue628",
	"ts":         "\ue628",
	"docker":     "\ue7b0",
	"github":     "\uf09b",
	"git":        "\uf1d3",
	"react":      "\ue7ba",
	"vue":        "\ue6d0",
	"angular":    "\ue753",
	"ruby":       "\ue739",
	"java":       "\ue738",
	"kotlin":     "\ue634",
	"swift":      "\ue755",
	"php":        "\ue73d",
	"c":          "\ue708",
	"cpp":        "\ue708",
	"c++":        "\ue708",
	"zig":        "\ue6a9",
	"deno":       "\ue60f",
	"discord":    "\uf392",
	"slack":      "\uf198",
	"nginx":      "\ue776",
	"redis":      "\ue76d",
	"postgresql": "\ue76e",
	"postgres":   "\ue76e",
	"mysql":      "\ue704",
	"mongodb":    "\ue7a4",
	"aws":        "\ue7ad",
	"amazon":     "\ue7ad",
	"linkedin":   "\uf0e1",
	"twitter":    "\uf099",
	"x":          "\ue619",
	"youtube":    "\uf167",
	"npm":        "\ue71e",
	"license":    "\uf0a3",
}
