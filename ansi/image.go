package ansi

import (
	"context"
	"fmt"
	"image"
	_ "image/jpeg" // required for JPEG decoding
	_ "image/png"  // required for PNG decoding
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/mosaic"
)

// pixelsPerCell is the approximate width of a terminal character cell in CSS pixels.
// Used to convert HTML <img width> attribute values to character cells.
const pixelsPerCell = 10

// An ImageElement is used to render images elements.
type ImageElement struct {
	Text     string
	BaseURL  string
	URL      string
	Child    ElementRenderer
	TextOnly bool
	Width    int
	Height   int
}

func loadImage(url string) (image.Image, error) {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		client := &http.Client{Timeout: 10 * time.Second}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("glamour: error creating request: %w", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("glamour: error fetching image: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, http.ErrMissingFile
		}
		img, _, err := image.Decode(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("glamour: error decoding image: %w", err)
		}
		return img, nil
	}
	f, err := os.Open(url)
	if err != nil {
		return nil, fmt.Errorf("glamour: error opening image file: %w", err)
	}
	defer func() { _ = f.Close() }()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("glamour: error decoding image: %w", err)
	}
	return img, nil
}

func (e *ImageElement) tryRenderMosaic(w io.Writer, ctx RenderContext) bool {
	if !ctx.options.MosaicEnabled || e.TextOnly {
		return false
	}
	u := resolveRelativeURL(e.BaseURL, e.URL)
	img, err := loadImage(u)
	if err != nil {
		return false
	}
	width := e.widthCells(ctx)
	if maxH := ctx.options.MosaicMaxHeight; maxH > 0 {
		width = scaleToMaxHeight(img, width, maxH)
	}
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	outW := width * 2
	outH := outW * srcH / srcW / 2
	if outH < 1 {
		outH = 1
	}
	m := mosaic.New().Width(outW).Height(outH)
	_, _ = io.WriteString(w, "\n"+m.Render(img))
	return true
}

func (e *ImageElement) widthCells(ctx RenderContext) int {
	if e.Width > 0 {
		w := e.Width / pixelsPerCell
		if w < 1 {
			w = 1
		}
		if ctx.options.MosaicWidth > 0 && w > ctx.options.MosaicWidth {
			w = ctx.options.MosaicWidth
		}
		return w
	}
	if ctx.options.MosaicWidth > 0 {
		return ctx.options.MosaicWidth
	}
	w := ctx.options.WordWrap / 2
	if w < 20 {
		w = 20
	}
	return w
}

// scaleToMaxHeight reduces width proportionally so the rendered image
// height does not exceed maxHeight. Uses source image aspect ratio.
func scaleToMaxHeight(img image.Image, widthCells, maxHeight int) int {
	b := img.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	if srcW <= 0 || srcH <= 0 {
		return widthCells
	}
	// Mosaic auto-height formula:
	//   outWidth = widthCells * 2
	//   outHeight = outWidth * srcH / srcW / 2  (divider=2 for cell aspect ratio)
	//   cellHeight = outHeight / 2 = widthCells * srcH / (2 * srcW)
	expH := widthCells * srcH / (2 * srcW)
	if expH > maxHeight {
		return widthCells * maxHeight / expH
	}
	return widthCells
}

func (e *ImageElement) tryRenderBadge(w io.Writer, ctx RenderContext) bool {
	if !ctx.options.ShieldsBadges || e.TextOnly {
		return false
	}
	u := resolveRelativeURL(e.BaseURL, e.URL)
	if !isShieldsURL(u) {
		return false
	}
	labelBg, fg := badgeThemeColors(ctx.options.Styles)
	// Try static badge (parse label/message/color from URL)
	if label, msg, colorStr, logo, ok := parseShieldsURL(u); ok {
		ansiColor := badgeNamedColor(colorStr)
		if len(colorStr) == 6 && isHex(colorStr) {
			ansiColor = hexToANSI(colorStr)
		}
		var icon string
		if ctx.options.NerdFontIcons {
			icon = logoNerdIcon(logo)
		}
		renderBadge(w, label, msg, ansiColor, labelBg, fg, icon)
		return true
	}
	// Dynamic badge: fetch SVG and extract info
	label, msg, ansiColor, ok := fetchShieldsBadge(u)
	if !ok {
		// Fallback: extract label from URL path
		label = shieldFallbackLabel(u)
		msg = ""
		if label == "" {
			return false
		}
		ansiColor = 32 // blue
	}
	renderBadge(w, label, msg, ansiColor, labelBg, fg, "")
	return true
}

func isHex(s string) bool {
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// Render renders an ImageElement.
func (e *ImageElement) Render(w io.Writer, ctx RenderContext) error {
	*ctx.hasParagraphImage = true
	if e.tryRenderBadge(w, ctx) {
		return nil
	}
	if e.tryRenderMosaic(w, ctx) {
		return nil
	}

	// Make OSC 8 hyperlink token.
	hyperlink, resetHyperlink, _ := makeHyperlink(e.URL)

	style := ctx.options.Styles.ImageText
	if e.TextOnly {
		style.Format = strings.TrimSuffix(style.Format, " →")
	}

	if len(e.Text) > 0 {
		token := hyperlink + e.Text + resetHyperlink
		el := &BaseElement{
			Token: token,
			Style: style,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	if e.TextOnly {
		return nil
	}

	if len(e.URL) > 0 {
		token := hyperlink + resolveRelativeURL(e.BaseURL, e.URL) + resetHyperlink
		el := &BaseElement{
			Token:  token,
			Prefix: " ",
			Style:  ctx.options.Styles.Image,
		}
		err := el.Render(w, ctx)
		if err != nil {
			return err
		}
	}

	_, _ = io.WriteString(w, "\n")
	return nil
}
