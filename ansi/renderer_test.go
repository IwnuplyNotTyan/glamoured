package ansi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/exp/golden"
	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

const (
	examplesDir = "../styles/examples/"
	issuesDir   = "../testdata/issues/"
)

func TestRenderer(t *testing.T) {
	files, err := filepath.Glob(examplesDir + "*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		bn := strings.TrimSuffix(filepath.Base(f), ".md")
		t.Run(bn, func(t *testing.T) {
			sn := filepath.Join(examplesDir, bn+".style")

			in, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(sn)
			if err != nil {
				t.Fatal(err)
			}

			options := Options{
				WordWrap: 80,
			}
			err = json.Unmarshal(b, &options.Styles)
			if err != nil {
				t.Fatal(err)
			}

			switch bn {
			case "table_wrap":
				tableWrap := true
				options.TableWrap = &tableWrap
			case "table_truncate":
				tableWrap := false
				options.TableWrap = &tableWrap
			case "table_with_inline_links":
				options.InlineTableLinks = true
			case "table_with_footer_links", "table_with_footer_links_no_color":
				options.InlineTableLinks = false
			}

			md := goldmark.New(
				goldmark.WithExtensions(
					extension.GFM,
					extension.DefinitionList,
					emoji.Emoji,
				),
				goldmark.WithParserOptions(
					parser.WithAutoHeadingID(),
					parser.WithASTTransformers(
						util.Prioritized(&CalloutMarkerTransformer{}, 100),
					),
				),
			)

			ar := NewRenderer(options)
			md.SetRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

			var buf bytes.Buffer
			if err := md.Convert(in, &buf); err != nil {
				t.Error(err)
			}

			golden.RequireEqual(t, buf.Bytes())
		})
	}
}

func TestRendererIssues(t *testing.T) {
	files, err := filepath.Glob(issuesDir + "*.md")
	if err != nil {
		t.Fatal(err)
	}

	for _, f := range files {
		bn := strings.TrimSuffix(filepath.Base(f), ".md")
		t.Run(bn, func(t *testing.T) {
			in, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile("../styles/dark.json")
			if err != nil {
				t.Fatal(err)
			}

			options := Options{
				WordWrap: 80,
			}
			err = json.Unmarshal(b, &options.Styles)
			if err != nil {
				t.Fatal(err)
			}
			if bn == "493" {
				tableWrap := false
				options.TableWrap = &tableWrap
			}

			md := goldmark.New(
				goldmark.WithExtensions(
					extension.GFM,
					extension.DefinitionList,
					emoji.Emoji,
				),
				goldmark.WithParserOptions(
					parser.WithAutoHeadingID(),
					parser.WithASTTransformers(
						util.Prioritized(&CalloutMarkerTransformer{}, 100),
					),
				),
			)

			ar := NewRenderer(options)
			md.SetRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

			var buf bytes.Buffer
			if err := md.Convert(in, &buf); err != nil {
				t.Error(err)
			}

			golden.RequireEqual(t, buf.Bytes())
		})
	}
}

func TestCalloutBlocks(t *testing.T) {
	input := `> [!NOTE]
> This is a note

> [!TIP]
> This is a tip

> [!IMPORTANT]
> This is important

> [!WARNING]
> This is a warning

> [!CAUTION]
> This is a caution

> [!NOTE] Inline note text
> More note text

> [!note] lowercase marker
> should still work
`

	options := Options{
		WordWrap: 80,
	}
	err := json.Unmarshal([]byte(`{
		"block_quote": {
			"indent": 1,
			"indent_token": "│ "
		}
	}`), &options.Styles)
	if err != nil {
		t.Fatal(err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			emoji.Emoji,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(&CalloutMarkerTransformer{}, 100),
			),
		),
	)

	ar := NewRenderer(options)
	md.SetRenderer(
		renderer.NewRenderer(
			renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if strings.Contains(output, "[!NOTE]") {
		t.Error("output should not contain raw [!NOTE] marker")
	}
	if strings.Contains(output, "[!TIP]") {
		t.Error("output should not contain raw [!TIP] marker")
	}
	if strings.Contains(output, "[!IMPORTANT]") {
		t.Error("output should not contain raw [!IMPORTANT] marker")
	}
	if strings.Contains(output, "[!WARNING]") {
		t.Error("output should not contain raw [!WARNING] marker")
	}
	if strings.Contains(output, "[!CAUTION]") {
		t.Error("output should not contain raw [!CAUTION] marker")
	}

	if !strings.Contains(output, "Note:") {
		t.Error("output should contain Note: label")
	}
	if !strings.Contains(output, "Tip:") {
		t.Error("output should contain Tip: label")
	}
	if !strings.Contains(output, "Important:") {
		t.Error("output should contain Important: label")
	}
	if !strings.Contains(output, "Warning:") {
		t.Error("output should contain Warning: label")
	}
	if !strings.Contains(output, "Caution:") {
		t.Error("output should contain Caution: label")
	}

	// Without NerdFontIcons, icons should not be present
	if strings.Contains(output, "\uf05a") {
		t.Error("output should not contain Nerd Font icon without NerdFontIcons option")
	}

	if !strings.Contains(output, "\x1b[38;5;39m") {
		t.Error("output should contain color 39 for NOTE indent")
	}
	if !strings.Contains(output, "\x1b[38;5;42m") {
		t.Error("output should contain color 42 for TIP indent")
	}
	if !strings.Contains(output, "\x1b[38;5;129m") {
		t.Error("output should contain color 129 for IMPORTANT indent")
	}
	if !strings.Contains(output, "\x1b[38;5;214m") {
		t.Error("output should contain color 214 for WARNING indent")
	}
	if !strings.Contains(output, "\x1b[38;5;196m") {
		t.Error("output should contain color 196 for CAUTION indent")
	}

	if !strings.Contains(output, "This is a note") {
		t.Error("output should contain 'This is a note'")
	}
	if !strings.Contains(output, "Inline note text") {
		t.Error("output should contain 'Inline note text'")
	}
	if !strings.Contains(output, "More note text") {
		t.Error("output should contain 'More note text'")
	}

	// Verify bold rendering for labels (bold is applied as part of combined SGR codes like \x1b[38;5;39;1m)
	if !strings.Contains(output, ";1m") && !strings.Contains(output, "\x1b[1m") {
		t.Error("output should contain bold escape sequence for labels")
	}
	if !strings.Contains(output, "lowercase marker") {
		t.Error("output should contain 'lowercase marker'")
	}
}

func TestCalloutBlocksWithNerdFontIcons(t *testing.T) {
	input := `> [!NOTE]
> Nerd note
`

	options := Options{
		WordWrap:      80,
		NerdFontIcons: true,
	}
	err := json.Unmarshal([]byte(`{
		"block_quote": {
			"indent": 1,
			"indent_token": "│ "
		}
	}`), &options.Styles)
	if err != nil {
		t.Fatal(err)
	}

	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.DefinitionList,
			emoji.Emoji,
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(&CalloutMarkerTransformer{}, 100),
			),
		),
	)

	ar := NewRenderer(options)
	md.SetRenderer(
		renderer.NewRenderer(
			renderer.WithNodeRenderers(util.Prioritized(ar, 1000))))

	var buf bytes.Buffer
	if err := md.Convert([]byte(input), &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()

	if !strings.Contains(output, "\uf05a") {
		t.Error("output should contain Nerd Font icon when NerdFontIcons is enabled")
	}
	if !strings.Contains(output, "Note:") {
		t.Error("output should contain Note: label")
	}
	if strings.Contains(output, "[!NOTE]") {
		t.Error("output should not contain raw marker")
	}
}

