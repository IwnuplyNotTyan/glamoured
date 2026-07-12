# Shields.io Badge Rendering

Render shields.io badge URLs as styled one-line ANSI badges with colored backgrounds and optional Nerd Font icons.

## Motivation

shields.io badges (e.g., `https://img.shields.io/badge/Go-1.21-blue`) are small SVGs that look terrible when rendered as mosaic art in the terminal. They should be detected and rendered as beautiful colored text badges instead.

## Design

### Detection

In `ImageElement.Render`, before attempting mosaic or text fallback, check if the image URL matches `img.shields.io`. If yes, render as a badge instead.

### Badge Format

One line, two colored segments:

```
▐ icon LABEL ▐▐ MESSAGE ▐
```

- **Label segment**: dark grey background (`\e[48;5;240m`), white text
- **Message segment**: badge color background, white text
- With **Nerd Font Icons** enabled: prepend the mapped icon before label text
- Without Nerd Font: plain text only

The badge is rendered as a standalone line (prepended with `\n`).

### URL Parsing

Parse static badge format: `https://img.shields.io/badge/<LABEL>-<MESSAGE>-<COLOR>`

Encoding rules:
- `--` → literal `-`
- `__` → literal `_`
- `_` → space ` `

Logo parameter from query string: `?logo=go` → looked up in Nerd Font mapping.

If parsing fails (e.g., dynamic badge URL), fall through to normal mosaic/text rendering.

### Colors

Support both:
1. **Named colors**: `brightgreen`, `green`, `yellowgreen`, `yellow`, `orange`, `red`, `blue`, `lightgrey`, `grey`, `blueviolet`, `pink`, `cyan`, `purple` — mapped to closest ANSI 256-color codes
2. **Hex colors**: 6-char hex (e.g., `007ec6`) — approximated to nearest ANSI 256-color

### Logo → Nerd Font Mapping

Map common shields.io logo names to Nerd Font Unicode codepoints. ~30 entries covering: go, rust, python, typescript, javascript, docker, github, git, node, react, aws, ruby, java, kotlin, swift, zig, php, c, deno, discord, slack, nginx, redis, postgresql, mongodb, vue, angular, haskell, elixir.

Unknown logos → generic badge icon (`\uf0a3` nf-fa-certificate or `\uf0a1` nf-fa-info-circle).

### API

New opt-out option:

```go
glamour.WithShieldsBadges(enabled bool) // default: true
```

When disabled, shields.io URLs are rendered as normal images (mosaic or text fallback).

### Options Struct

```go
type Options struct {
    // ... existing fields
    ShieldsBadges bool
}
```

### Rendering Location

New method `tryRenderBadge(w, ctx)` on `ImageElement`:

1. Called at the start of `Render()`, before `tryRenderMosaic()`
2. Returns `true` if successfully rendered a badge
3. Renders ANSI escape sequences directly to writer

### What NOT to Do

- No `<img width/height>` support for badges (irrelevant)
- No dynamic/endpoint badges (too complex; fall through to normal rendering)
- No text fallback for badges (if URL parse fails, fall through to normal rendering)
- No query param style parsing (always flat/simple style)
