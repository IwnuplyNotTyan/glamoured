package ansi

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/x/mosaic"
)

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
		resp, err := client.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, http.ErrMissingFile
		}
		img, _, err := image.Decode(resp.Body)
		return img, err
	}
	f, err := os.Open(url)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// Render renders an ImageElement.
func (e *ImageElement) Render(w io.Writer, ctx RenderContext) error {
	if ctx.options.MosaicEnabled && !e.TextOnly {
		u := resolveRelativeURL(e.BaseURL, e.URL)
		img, err := loadImage(u)
		if err == nil {
			width := e.Width
			if width <= 0 {
				width = ctx.options.MosaicWidth
			}
			if width <= 0 {
				width = ctx.options.WordWrap / 2
				if width < 20 {
					width = 20
				}
			}
<<<<<<< HEAD
=======
			pixelW := width * 2
			b := img.Bounds()
			srcW := b.Max.X - b.Min.X
			srcH := b.Max.Y - b.Min.Y
			pixelH := int(float64(pixelW) * float64(srcH) / float64(srcW) / 2)
			if e.Height > 0 {
				pixelH = e.Height * 2
			}
			if pixelH < 1 {
				pixelH = 1
			}
>>>>>>> 2f75868 (feat: add Width/Height fields to ImageElement and parse <img> HTML tags)
			m := mosaic.New()
			m = m.Width(width * 2)
			art := m.Render(img)
			el := &BaseElement{
				Token: art,
				Style: ctx.options.Styles.Image,
			}
			return el.Render(w, ctx)
		}
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

	return nil
}
