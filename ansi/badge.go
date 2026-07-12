package ansi

import (
	"net/url"
	"strings"
)

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
	// Protect literal dashes (--) before splitting on single dash
	path = strings.ReplaceAll(path, "--", "\x00")
	parts := strings.Split(path, "-")
	// Restore dashes in each part
	for i := range parts {
		parts[i] = strings.ReplaceAll(parts[i], "\x00", "-")
	}
	if len(parts) < 3 {
		return "", "", "", "", false
	}
	color = parts[len(parts)-1]
	message = parts[len(parts)-2]
	label = strings.Join(parts[:len(parts)-2], "-")
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
