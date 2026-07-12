package glamour

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
	"github.com/iwnuplynottyan/glamoured/styles"
)

const markdown = "testdata/readme.markdown.in"

func TestTermRendererWriter(t *testing.T) {
	r, err := NewTermRenderer(
		WithStandardStyle(styles.DarkStyle),
	)
	if err != nil {
		t.Fatal(err)
	}

	in, err := os.ReadFile(markdown)
	if err != nil {
		t.Fatal(err)
	}

	_, err = r.Write(in)
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

	golden.RequireEqual(t, b)
}

func TestTermRenderer(t *testing.T) {
	r, err := NewTermRenderer(
		WithStandardStyle("dark"),
	)
	if err != nil {
		t.Fatal(err)
	}

	in, err := os.ReadFile(markdown)
	if err != nil {
		t.Fatal(err)
	}

	b, err := r.Render(string(in))
	if err != nil {
		t.Fatal(err)
	}

	golden.RequireEqual(t, []byte(b))
}

func TestWithEmoji(t *testing.T) {
	r, err := NewTermRenderer(
		WithStandardStyle("notty"),
		WithEmoji(),
	)
	if err != nil {
		t.Fatal(err)
	}

	b, err := r.Render(":+1:")
	if err != nil {
		t.Fatal(err)
	}
	b = strings.TrimSpace(b)

	// Thumbs up unicode character
	td := "\U0001f44d"

	if td != b {
		t.Errorf("Rendered output doesn't match!\nExpected: `\n%s`\nGot: `\n%s`\n", td, b)
	}
}

func TestWithPreservedNewLines(t *testing.T) {
	r, err := NewTermRenderer(
		WithPreservedNewLines(),
	)
	if err != nil {
		t.Fatal(err)
	}

	in, err := os.ReadFile("testdata/preserved_newline.in")
	if err != nil {
		t.Fatal(err)
	}

	b, err := r.Render(string(in))
	if err != nil {
		t.Fatal(err)
	}

	golden.RequireEqual(t, []byte(b))
}

func TestStyles(t *testing.T) {
	_, err := NewTermRenderer()
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewTermRenderer(
		WithStandardStyle(styles.DarkStyle),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = NewTermRenderer(
		WithEnvironmentConfig(),
	)
	if err != nil {
		t.Fatal(err)
	}
}

// TestCustomStyle checks the expected errors with custom styling. We need to
// support built-in styles and custom style sheets.
func TestCustomStyle(t *testing.T) {
	md := "testdata/example.md"
	tests := []struct {
		name      string
		stylePath string
		err       error
		expected  string
	}{
		{name: "style exists", stylePath: "testdata/custom.style", err: nil, expected: "testdata/custom.style"},
		{name: "style doesn't exist", stylePath: "testdata/notfound.style", err: os.ErrNotExist, expected: styles.DarkStyle},
		{name: "style is empty", stylePath: "", err: nil, expected: styles.DarkStyle},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GLAMOUR_STYLE", tc.stylePath)
			g, err := NewTermRenderer(
				WithEnvironmentConfig(),
			)
			if !errors.Is(err, tc.err) {
				t.Fatal(err)
			}
			if !errors.Is(tc.err, os.ErrNotExist) {
				w, err := NewTermRenderer(WithStylePath(tc.expected))
				if err != nil {
					t.Fatal(err)
				}
				text, _ := os.ReadFile(md)
				want, err := w.RenderBytes(text)
				got, err := g.RenderBytes(text)
				if !bytes.Equal(want, got) {
					t.Error("Wrong style used")
				}
			}
		})
	}
}

func TestRenderHelpers(t *testing.T) {
	in, err := os.ReadFile(markdown)
	if err != nil {
		t.Fatal(err)
	}

	b, err := Render(string(in), "dark")
	if err != nil {
		t.Error(err)
	}

	golden.RequireEqual(t, []byte(b))
}

func TestCapitalization(t *testing.T) {
	p := true
	style := styles.DarkStyleConfig
	style.H1.Upper = &p
	style.H2.Title = &p
	style.H3.Lower = &p

	r, err := NewTermRenderer(
		WithStyles(style),
	)
	if err != nil {
		t.Fatal(err)
	}

	b, err := r.Render("# everything is uppercase\n## everything is titled\n### everything is lowercase")
	if err != nil {
		t.Fatal(err)
	}

	golden.RequireEqual(t, []byte(b))
}

func FuzzData(f *testing.F) {
	f.Fuzz(func(t *testing.T, data []byte) {
		func() int {
			_, err := RenderBytes(data, styles.DarkStyle)
			if err != nil {
				return 0
			}
			return 1
		}()
	})
}

func TestTableAscii(t *testing.T) {
	markdown := strings.TrimSpace(`
| Header A  | Header B  |
| --------- | --------- |
| Cell 1    | Cell 2    |
| Cell 3    | Cell 4    |
| Cell 5    | Cell 6    |
`)

	renderer, err := NewTermRenderer(
		WithStyles(styles.ASCIIStyleConfig),
		WithWordWrap(80),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := renderer.Render(markdown)
	if err != nil {
		t.Fatal(err)
	}

	nonAsciiRegexp := regexp.MustCompile(`[^\x00-\x7f]+`)
	nonAsciiChars := nonAsciiRegexp.FindAllString(result, -1)
	if len(nonAsciiChars) > 0 {
		t.Errorf("Non-ASCII characters found in output: %v", nonAsciiChars)
	}
}

func ExampleASCIIStyleConfig() {
	markdown := strings.TrimSpace(`
| Header A  | Header B  |
| --------- | --------- |
| Cell 1    | Cell 2    |
| Cell 3    | Cell 4    |
| Cell 5    | Cell 6    |
`)

	renderer, err := NewTermRenderer(
		WithStyles(styles.ASCIIStyleConfig),
		WithWordWrap(80),
	)
	if err != nil {
		return
	}

	result, err := renderer.Render(markdown)
	if err != nil {
		return
	}
	result = strings.ReplaceAll(result, " ", ".")
	fmt.Println(result)

	// Output:
	// ..............................................................................
	// ...Header.A.............................|.Header.B............................
	// ..--------------------------------------|-------------------------------------
	// ...Cell.1...............................|.Cell.2..............................
	// ...Cell.3...............................|.Cell.4..............................
	// ...Cell.5...............................|.Cell.6..............................
}

func TestWithShieldsBadges(t *testing.T) {
	r, err := NewTermRenderer()
	if err != nil {
		t.Fatal(err)
	}
	if !r.ansiOptions.ShieldsBadges {
		t.Error("expected ShieldsBadges to default to true")
	}

	r2, err := NewTermRenderer(WithShieldsBadges(false))
	if err != nil {
		t.Fatal(err)
	}
	if r2.ansiOptions.ShieldsBadges {
		t.Error("expected ShieldsBadges to be false")
	}
}

func TestWithChromaFormatterDefault(t *testing.T) {
	r, err := NewTermRenderer(
		WithStandardStyle(styles.DarkStyle),
	)
	if err != nil {
		t.Fatal(err)
	}

	in, err := os.ReadFile("testdata/TestWithChromaFormatter.md")
	if err != nil {
		t.Fatal(err)
	}

	b, err := r.Render(string(in))
	if err != nil {
		t.Fatal(err)
	}

	golden.RequireEqual(t, []byte(b))
}

func TestCenterBlock(t *testing.T) {
	tests := []struct {
		name  string
		md    string
		check string
	}{
		{
			name:  "center tag",
			md:    "<center>\n**Bold**\n</center>",
			check: "Bold",
		},
		{
			name:  "div align center",
			md:    "<div align=\"center\">\n**Bold**\n</div>",
			check: "Bold",
		},
		{
			name:  "div align center unquoted",
			md:    "<div align=center>\n**Bold**\n</div>",
			check: "Bold",
		},
		{
			name:  "center with image",
			md:    "<center>\n![alt](https://example.com/img.png)\n</center>",
			check: "alt",
		},
		{
			name:  "center in document",
			md:    "# Title\n\n<center>\n**Centered**\n</center>\n\nFooter",
			check: "Centered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewTermRenderer(
				WithWordWrap(80),
				WithStandardStyle("dark"),
			)
			if err != nil {
				t.Fatal(err)
			}

			out, err := r.RenderBytes([]byte(tt.md))
			if err != nil {
				t.Fatal(err)
			}

			clean := stripANSI(string(out))

			if strings.Contains(clean, "\x00") {
				t.Error("marker bytes found in output")
			}

			if !strings.Contains(clean, tt.check) {
				t.Errorf("expected output to contain %q", tt.check)
			}
		})
	}
}

func TestWithChromaFormatterCustom(t *testing.T) {
	r, err := NewTermRenderer(
		WithStandardStyle(styles.DarkStyle),
		WithChromaFormatter("terminal16"),
	)
	if err != nil {
		t.Fatal(err)
	}

	in, err := os.ReadFile("testdata/TestWithChromaFormatter.md")
	if err != nil {
		t.Fatal(err)
	}

	b, err := r.Render(string(in))
	if err != nil {
		t.Fatal(err)
	}

	golden.RequireEqual(t, []byte(b))
}
