# Glamoured - Glamour but better

<p>
    <a href="https://github.com/iwnuplynottyan/glamoured/actions"><img src="https://github.com/iwnuplynottyan/glamour/workflows/build/badge.svg" alt="Build Status"></a>
    <a href="https://coveralls.io/github/iwnuplynottyan/glamour?branch=master"><img src="https://coveralls.io/repos/github/iwnuplynottyan/glamour/badge.svg?branch=master" alt="Coverage Status"></a>
</p>

Stylesheet-based markdown rendering for your CLI apps.

`glamour` lets you render markdown documents & templates on ANSI
compatible terminals. You can create your own stylesheet or simply use one of
the stylish defaults.

This is an enhanced fork with additional features beyond the [original](https://github.com/charmbracelet/glamour/).

## Enhanced Features

### Mosaic Image Rendering

Render images as ANSI mosaic art directly in the terminal:

```go
r, _ := glamour.NewTermRenderer(
    glamour.WithStandardStyle("dark"),
    glamour.WithMosaic(true),
    glamour.WithMosaicWidth(40), // character cells width
)

out, _ := r.Render("![image](https://example.com/image.png)")
fmt.Print(out)
```

- `WithMosaic(enabled bool)` — enable/disable (default: enabled)
- `WithMosaicWidth(width int)` — set max width in character cells (default: half of WordWrap)

### HTML `<center>` and `<div align="center">`

Center content horizontally. Content inside is rendered as markdown:

```markdown
<center>
**Bold centered text**

![image](example.png)
</center>

<div align="center">
Centered via div
</div>
```

Both `<center>` and `<div align="center">` (with or without quotes) are supported.

### HTML `<img>` Width and Height

Override image rendering dimensions via HTML attributes:

```markdown
<img src="image.png" width="40" height="30">
```

Width and height are in character cells. When set, they override `MosaicWidth` and
the automatic aspect-ratio height calculation.

### GitHub-Style Callout Blocks

Render `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`
as styled blockquote callouts with colored labels:

```markdown
> [!NOTE]
> Useful information for the user.

> [!WARNING]
> Be careful!
```

Output shows a colored `│` prefix and a bold colored label (e.g., `Note:`, `Warning:`).

#### Nerd Font Icons

To show Nerd Font glyphs instead of plain labels, enable the option:

```go
r, _ := glamour.NewTermRenderer(
    glamour.WithStandardStyle("dark"),
    glamour.WithNerdFontIcons(),
)
```

This renders `  Note:`, `  Tip:`, `  Important:`, `  Warning:`, `  Caution:`
(requires a [Nerd Font](https://www.nerdfonts.com/) in your terminal).

## Base Usage

```go
import "github.com/iwnuplynottyan/glamoured"

in := `# Hello World

This is a simple example of Markdown rendering with Glamour!
Check out the [other examples](https://github.com/charmbracelet/glamour/tree/main/examples) too.

Bye!
`

out, err := glamour.Render(in, "dark")
fmt.Print(out)
```

### Custom Renderer

```go
import "github.com/iwnuplynottyan/glamoured"

r, _ := glamour.NewTermRenderer(
    // wrap output at specific width (default is 80)
    glamour.WithWordWrap(40),
)

out, err := r.Render(in)
fmt.Print(out)
```

### Custom Renderer Options

| Option | Description |
|--------|-------------|
| `WithWordWrap(n)` | Set word wrap width (default 80) |
| `WithStandardStyle(name)` | Use a built-in style ("dark", "light", "pink", etc.) |
| `WithStylePath(path)` | Use a custom JSON style file |
| `WithStyles(styleConfig)` | Use a programmatic `StyleConfig` |
| `WithMosaic(enabled)` | Enable/disable mosaic image rendering |
| `WithMosaicWidth(width)` | Set mosaic image max width in cells |
| `WithNerdFontIcons()` | Enable Nerd Font icons for callout blocks |
| `WithEmoji()` | Enable emoji rendering (`:+1:` → 👍) |
| `WithPreservedNewLines()` | Preserve newlines in output |
| `WithBaseURL(url)` | Set base URL for relative links |
| `WithChromaFormatter(name)` | Set syntax highlighting formatter |

### Color Downsampling

Since the renderer is designed to be "pure" and always produce the same output
for the same input, it doesn't have access to the terminal's capabilities. This
means that color downsampling is not performed by default. Use [Lip Gloss][lipgloss]
to perform downsampling before rendering:

```go
import (
    "github.com/iwnuplynottyan/glamoured"
    "charm.land/lipgloss/v2"
)

r, _ := glamour.NewTermRenderer(
    glamour.WithWordWrap(40),
)

out, err := r.Render(in)
if err != nil {
    // handle error
}

// downsample colors based on terminal capabilities.
lipgloss.Print(out)
```

[lipgloss]: https://github.com/charmbracelet/lipgloss

## Styles

You can find all available default styles in our [gallery](https://github.com/iwnuplynottyan/glamour/tree/main/styles/gallery).
Want to create your own style? [Learn how!](https://github.com/iwnuplynottyan/glamoured/tree/main/styles)

There are a few options for using a custom style:

1. Call `glamour.Render(inputText, "desiredStyle")`
1. Set the `GLAMOUR_STYLE` environment variable to your desired default style or a file location for a style and call `glamour.RenderWithEnvironmentConfig(inputText)`
1. Set the `GLAMOUR_STYLE` environment variable and pass `glamour.WithEnvironmentConfig()` to your custom renderer

## Contributing

See [contributing][contribute].

[contribute]: https://github.com/charmbracelet/glamour/contribute

## License

[MIT](https://github.com/charmbracelet/glamour/raw/master/LICENSE)

---

<div align="center">
  <h1>Made with ❤️ </h1>
</div>
