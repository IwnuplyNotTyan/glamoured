package ansi

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"

	"charm.land/glamour/v2/internal/autolink"
	east "github.com/yuin/goldmark-emoji/ast"
	"github.com/yuin/goldmark/ast"
	astext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	nhtml "golang.org/x/net/html"
)

// ElementRenderer is called when entering a markdown node.
type ElementRenderer interface {
	Render(w io.Writer, ctx RenderContext) error
}

// StyleOverriderElementRenderer is called when entering a markdown node with a specific style.
type StyleOverriderElementRenderer interface {
	StyleOverrideRender(w io.Writer, ctx RenderContext, style StylePrimitive) error
}

// ElementFinisher is called when leaving a markdown node.
type ElementFinisher interface {
	Finish(w io.Writer, ctx RenderContext) error
}

// An Element is used to instruct the renderer how to handle individual markdown
// nodes.
type Element struct {
	Entering string
	Exiting  string
	Renderer ElementRenderer
	Finisher ElementFinisher
}

// NewElement returns the appropriate render Element for a given node.
func (tr *ANSIRenderer) NewElement(node ast.Node, source []byte) Element {
	ctx := tr.context

	switch node.Kind() {
	// Document
	case ast.KindDocument:
		e := &BlockElement{
			Block:  &bytes.Buffer{},
			Style:  ctx.options.Styles.Document,
			Margin: true,
		}
		return Element{
			Renderer: e,
			Finisher: e,
		}

	// Heading
	case ast.KindHeading:
		n := node.(*ast.Heading)
		he := &HeadingElement{
			Level: n.Level,
			First: node.PreviousSibling() == nil,
		}
		return Element{
			Exiting:  "",
			Renderer: he,
			Finisher: he,
		}

	// Paragraph
	case ast.KindParagraph:
		if node.Parent() != nil {
			kind := node.Parent().Kind()
			if kind == ast.KindListItem {
				return Element{}
			}
		}
		return Element{
			Renderer: &ParagraphElement{
				First: node.PreviousSibling() == nil,
			},
			Finisher: &ParagraphElement{},
		}

	// Blockquote
	case ast.KindBlockquote:
		e := &BlockElement{
			Block:        &bytes.Buffer{},
			Style:        cascadeStyle(ctx.blockStack.Current().Style, ctx.options.Styles.BlockQuote, false),
			Margin:       true,
			IsBlockquote: true,
		}
		return Element{
			Entering: "\n",
			Renderer: e,
			Finisher: e,
		}

	// Lists
	case ast.KindList:
		s := ctx.options.Styles.List.StyleBlock
		if s.Indent == nil {
			var i uint
			s.Indent = &i
		}
		n := node.Parent()
		for n != nil {
			if n.Kind() == ast.KindList {
				i := ctx.options.Styles.List.LevelIndent
				s.Indent = &i
				break
			}
			n = n.Parent()
		}

		e := &BlockElement{
			Block:   &bytes.Buffer{},
			Style:   cascadeStyle(ctx.blockStack.Current().Style, s, false),
			Margin:  true,
			Newline: true,
		}
		return Element{
			Entering: "\n",
			Renderer: e,
			Finisher: e,
		}

	case ast.KindListItem:
		var l uint
		var e uint
		l = 1
		n := node
		for n.PreviousSibling() != nil && (n.PreviousSibling().Kind() == ast.KindListItem) {
			l++
			n = n.PreviousSibling()
		}
		if node.Parent().(*ast.List).IsOrdered() {
			e = l
			if node.Parent().(*ast.List).Start != 1 {
				e += uint(node.Parent().(*ast.List).Start) - 1
			}
		}

		post := "\n"
		if (node.LastChild() != nil && node.LastChild().Kind() == ast.KindList) ||
			node.NextSibling() == nil {
			post = ""
		}

		if node.FirstChild() != nil &&
			node.FirstChild().FirstChild() != nil &&
			node.FirstChild().FirstChild().Kind() == astext.KindTaskCheckBox {
			nc := node.FirstChild().FirstChild().(*astext.TaskCheckBox)

			return Element{
				Exiting: post,
				Renderer: &TaskElement{
					Checked: nc.IsChecked,
				},
			}
		}

		return Element{
			Exiting: post,
			Renderer: &ItemElement{
				IsOrdered:   node.Parent().(*ast.List).IsOrdered(),
				Enumeration: e,
			},
		}

	// Text Elements
	case ast.KindText:
		n := node.(*ast.Text)
		s := string(n.Segment.Value(source))

		if ct := detectCallout(s); ct != nil && isInsideBlockquote(node) {
			colorBlockquoteIndent(ctx, ct.color)
			rest := strings.TrimLeft(s[len(ct.marker):], " ")

			if n.HardLineBreak() || n.SoftLineBreak() {
				rest += "\n"
			}

			return Element{
				Renderer: &CalloutElement{
					Label:    ct.label + " ",
					NerdIcon: ct.nerdIcon,
					Rest:     rest,
					Color:    ct.color,
				},
			}
		}

		if n.HardLineBreak() || (n.SoftLineBreak()) {
			s += "\n"
		}
		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(s),
				Style: ctx.options.Styles.Text,
			},
		}

	case ast.KindEmphasis:
		n := node.(*ast.Emphasis)
		var children []ElementRenderer
		nn := n.FirstChild()
		for nn != nil {
			children = append(children, tr.NewElement(nn, source).Renderer)
			nn = nn.NextSibling()
		}
		return Element{
			Renderer: &EmphasisElement{
				Level:    n.Level,
				Children: children,
			},
		}

	case astext.KindStrikethrough:
		n := node.(*astext.Strikethrough)
		s := string(n.Text(source)) //nolint: staticcheck
		style := ctx.options.Styles.Strikethrough

		return Element{
			Renderer: &BaseElement{
				Token: html.UnescapeString(s),
				Style: style,
			},
		}

	case ast.KindThematicBreak:
		return Element{
			Entering: "",
			Exiting:  "",
			Renderer: &BaseElement{
				Style: ctx.options.Styles.HorizontalRule,
			},
		}

	// Links
	case ast.KindLink:
		n := node.(*ast.Link)
		isFooterLinks := !ctx.options.InlineTableLinks && isInsideTable(node)

		var children []ElementRenderer
		content, err := nodeContent(node, source)

		if isFooterLinks && err == nil {
			text := string(content)
			tl := tableLink{
				content:  text,
				href:     string(n.Destination),
				title:    string(n.Title),
				linkType: linkTypeRegular,
			}
			text = linkWithSuffix(tl, ctx.table.tableLinks)
			children = []ElementRenderer{&BaseElement{Token: text}}
		} else {
			nn := n.FirstChild()
			for nn != nil {
				children = append(children, tr.NewElement(nn, source).Renderer)
				nn = nn.NextSibling()
			}
		}

		return Element{
			Renderer: &LinkElement{
				BaseURL:  ctx.options.BaseURL,
				URL:      string(n.Destination),
				Children: children,
				SkipHref: isFooterLinks,
			},
		}
	case ast.KindAutoLink:
		n := node.(*ast.AutoLink)
		u := string(n.URL(source))
		isFooterLinks := !ctx.options.InlineTableLinks && isInsideTable(node)

		var children []ElementRenderer
		nn := n.FirstChild()
		for nn != nil {
			children = append(children, tr.NewElement(nn, source).Renderer)
			nn = nn.NextSibling()
		}

		if len(children) == 0 {
			children = append(children, &BaseElement{Token: u})
		}

		if n.AutoLinkType == ast.AutoLinkEmail && !strings.HasPrefix(strings.ToLower(u), "mailto:") {
			u = "mailto:" + u
		}

		var renderer ElementRenderer
		if isFooterLinks {
			domain := linkDomain(u)
			tl := tableLink{
				content:  domain,
				href:     u,
				linkType: linkTypeAuto,
			}
			if shortned, ok := autolink.Detect(u); ok {
				tl.content = shortned
			}
			text := linkWithSuffix(tl, ctx.table.tableLinks)

			renderer = &LinkElement{
				Children: []ElementRenderer{&BaseElement{Token: text}},
				URL:      u,
				SkipHref: true,
			}
		} else {
			isEmail := n.AutoLinkType == ast.AutoLinkEmail
			renderer = &LinkElement{
				Children: children,
				URL:      u,
				SkipText: !isEmail, // For non-email links, skip text (show only href)
				SkipHref: isEmail,  // For email links, skip href (hide mailto: URL)
			}
		}
		return Element{Renderer: renderer}

	// Images
	case ast.KindImage:
		n := node.(*ast.Image)
		text := string(n.Text(source)) //nolint: staticcheck
		isFooterLinks := !ctx.options.InlineTableLinks && isInsideTable(node)

		if isFooterLinks {
			if text == "" {
				text = linkDomain(string(n.Destination))
			}
			tl := tableLink{
				title:    string(n.Title),
				content:  text,
				href:     string(n.Destination),
				linkType: linkTypeImage,
			}
			text = linkWithSuffix(tl, ctx.table.tableImages)
		}

		return Element{
			Renderer: &ImageElement{
				Text:     text,
				BaseURL:  ctx.options.BaseURL,
				URL:      string(n.Destination),
				TextOnly: isFooterLinks,
			},
		}

	// Code
	case ast.KindFencedCodeBlock:
		n := node.(*ast.FencedCodeBlock)
		l := n.Lines().Len()
		s := ""
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			s += string(line.Value(source))
		}
		return Element{
			Entering: "\n",
			Renderer: &CodeBlockElement{
				Code:     s,
				Language: string(n.Language(source)),
			},
		}

	case ast.KindCodeBlock:
		n := node.(*ast.CodeBlock)
		l := n.Lines().Len()
		s := ""
		for i := 0; i < l; i++ {
			line := n.Lines().At(i)
			s += string(line.Value(source))
		}
		return Element{
			Entering: "\n",
			Renderer: &CodeBlockElement{
				Code: s,
			},
		}

	case ast.KindCodeSpan:
		n := node.(*ast.CodeSpan)
		s := string(n.Text(source)) //nolint: staticcheck
		return Element{
			Renderer: &CodeSpanElement{
				Text:  html.UnescapeString(s),
				Style: cascadeStyle(ctx.blockStack.Current().Style, ctx.options.Styles.Code, false).StylePrimitive,
			},
		}

	// Tables
	case astext.KindTable:
		table := node.(*astext.Table)
		te := &TableElement{
			table:  table,
			source: source,
		}
		return Element{
			Entering: "\n",
			Exiting:  "\n",
			Renderer: te,
			Finisher: te,
		}

	case astext.KindTableCell:
		n := node.(*astext.TableCell)
		var children []ElementRenderer
		nn := n.FirstChild()
		for nn != nil {
			children = append(children, tr.NewElement(nn, source).Renderer)
			nn = nn.NextSibling()
		}

		r := &TableCellElement{
			Children: children,
			Head:     node.Parent().Kind() == astext.KindTableHeader,
		}
		return Element{
			Renderer: r,
		}

	case astext.KindTableHeader:
		return Element{
			Finisher: &TableHeadElement{},
		}
	case astext.KindTableRow:
		return Element{
			Finisher: &TableRowElement{},
		}

	// HTML Elements
	case ast.KindHTMLBlock:
		n := node.(*ast.HTMLBlock)
		raw := string(n.Text(source))
		if src, w, h := parseHTMLImage(raw); src != "" {
			return Element{
				Renderer: &ImageElement{
					BaseURL:  ctx.options.BaseURL,
					URL:      src,
					TextOnly: false,
					Width:    w,
					Height:   h,
				},
			}
		}
		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(raw, true), //nolint: staticcheck
				Style: ctx.options.Styles.HTMLBlock.StylePrimitive,
			},
		}
	case ast.KindRawHTML:
		n := node.(*ast.RawHTML)
		raw := string(n.Text(source))
		if src, w, h := parseHTMLImage(raw); src != "" {
			return Element{
				Renderer: &ImageElement{
					BaseURL:  ctx.options.BaseURL,
					URL:      src,
					TextOnly: false,
					Width:    w,
					Height:   h,
				},
			}
		}
		return Element{
			Renderer: &BaseElement{
				Token: ctx.SanitizeHTML(raw, true), //nolint: staticcheck
				Style: ctx.options.Styles.HTMLSpan.StylePrimitive,
			},
		}

	// Definition Lists
	case astext.KindDefinitionList:
		e := &BlockElement{
			Block:   &bytes.Buffer{},
			Style:   cascadeStyle(ctx.blockStack.Current().Style, ctx.options.Styles.DefinitionList, false),
			Margin:  true,
			Newline: true,
		}
		return Element{
			Renderer: e,
			Finisher: e,
		}

	case astext.KindDefinitionTerm:
		return Element{
			Entering: "\n",
			Renderer: &BaseElement{
				Style: ctx.options.Styles.DefinitionTerm,
			},
		}

	case astext.KindDefinitionDescription:
		return Element{
			Exiting: "\n",
			Renderer: &BaseElement{
				Style: ctx.options.Styles.DefinitionDescription,
			},
		}

	// Handled by parents
	case astext.KindTaskCheckBox:
		// handled by KindListItem
		return Element{}
	case ast.KindTextBlock:
		return Element{}

	case east.KindEmoji:
		n := node.(*east.Emoji)
		return Element{
			Renderer: &BaseElement{
				Token: string(n.Value.Unicode),
			},
		}

	// Unknown case
	default:
		fmt.Println("Warning: unhandled element", node.Kind().String())
		return Element{}
	}
}

// parseHTMLImage parses an HTML string looking for <img> tags.
// Returns the src, width, height if found, otherwise empty strings/0.
func parseHTMLImage(htmlInput string) (src string, width int, height int) {
	doc, err := nhtml.Parse(strings.NewReader(htmlInput))
	if err != nil {
		return
	}
	var findImg func(*nhtml.Node) bool
	findImg = func(n *nhtml.Node) bool {
		if n.Type == nhtml.ElementNode && n.Data == "img" {
			for _, a := range n.Attr {
				switch a.Key {
				case "src":
					src = a.Val
				case "width":
					width, _ = strconv.Atoi(a.Val)
				case "height":
					height, _ = strconv.Atoi(a.Val)
				}
			}
			return true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if findImg(c) {
				return true
			}
		}
		return false
	}
	findImg(doc)
	return
}

type calloutType struct {
	marker   string
	label    string
	color    string
	nerdIcon string
}

var calloutTypes = []calloutType{
	{marker: "[!NOTE]", label: "Note:", color: "39", nerdIcon: "\uf05a"},
	{marker: "[!TIP]", label: "Tip:", color: "42", nerdIcon: "\uf0eb"},
	{marker: "[!IMPORTANT]", label: "Important:", color: "129", nerdIcon: "\uf06a"},
	{marker: "[!WARNING]", label: "Warning:", color: "214", nerdIcon: "\uf071"},
	{marker: "[!CAUTION]", label: "Caution:", color: "196", nerdIcon: "\uf057"},
}

func detectCallout(s string) *calloutType {
	upper := strings.ToUpper(s)
	for _, ct := range calloutTypes {
		if strings.HasPrefix(upper, ct.marker) {
			c := ct
			return &c
		}
	}
	return nil
}

func isInsideBlockquote(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == ast.KindBlockquote {
			return true
		}
	}
	return false
}

func colorBlockquoteIndent(ctx RenderContext, color string) {
	bs := *ctx.blockStack
	for i := len(bs) - 1; i >= 0; i-- {
		if bs[i].IsBlockquote {
			token := " "
			if bs[i].Style.IndentToken != nil {
				token = *bs[i].Style.IndentToken
			}
			styledToken := fmt.Sprintf("\x1b[38;5;%sm%s\x1b[0m", color, token)
			bs[i].Style.IndentToken = &styledToken
			return
		}
	}
}

// CalloutElement renders a GitHub-style callout label.
type CalloutElement struct {
	Label    string
	NerdIcon string
	Rest     string
	Color    string
}

// Render renders the callout label with styled text.
func (e *CalloutElement) Render(w io.Writer, ctx RenderContext) error {
	bs := ctx.blockStack

	labelText := e.Label
	if ctx.options.NerdFontIcons && e.NerdIcon != "" {
		labelText = e.NerdIcon + " " + e.Label
	}

	labelStyle := StylePrimitive{
		Color: &e.Color,
		Bold:  boolPtr(true),
	}
	labelStyle = cascadeStylePrimitives(bs.Current().Style.StylePrimitive, labelStyle)
	if _, err := renderText(w, labelStyle, labelText); err != nil {
		return err
	}

	if e.Rest != "" {
		restStyle := cascadeStylePrimitives(bs.Current().Style.StylePrimitive, ctx.options.Styles.Text)
		if _, err := renderText(w, restStyle, escapeReplacer.Replace(html.UnescapeString(e.Rest))); err != nil {
			return err
		}
	}

	return nil
}

func boolPtr(b bool) *bool { return &b }

// CalloutMarkerTransformer transforms goldmark AST to merge callout markers
// split by the parser into separate text nodes.
type CalloutMarkerTransformer struct{}

func (t *CalloutMarkerTransformer) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	src := reader.Source()
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || n.Kind() != ast.KindParagraph {
			return ast.WalkContinue, nil
		}
		if n.Parent() == nil || n.Parent().Kind() != ast.KindBlockquote {
			return ast.WalkContinue, nil
		}

		if n.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		seg := n.Lines().At(0)
		firstLine := string(seg.Value(src))

		var matchedMarker string
		upper := strings.ToUpper(firstLine)
		for _, ct := range calloutTypes {
			if strings.HasPrefix(upper, ct.marker) {
				matchedMarker = ct.marker
				break
			}
		}
		if matchedMarker == "" {
			return ast.WalkContinue, nil
		}

		var textNodes []*ast.Text
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				textNodes = append(textNodes, child.(*ast.Text))
			}
		}
		if len(textNodes) < 2 {
			return ast.WalkContinue, nil
		}

		var acc string
		mergeCount := 0
		for _, tn := range textNodes {
			acc += string(tn.Segment.Value(src))
			mergeCount++
			if len(acc) >= len(matchedMarker) {
				break
			}
		}
		if mergeCount <= 1 {
			return ast.WalkContinue, nil
		}

		first := textNodes[0]
		last := textNodes[mergeCount-1]

		if last.Segment.Stop > first.Segment.Stop {
			first.Segment.Stop = last.Segment.Stop
		}
		if last.SoftLineBreak() {
			first.SetSoftLineBreak(true)
		}

		for i := mergeCount - 1; i >= 1; i-- {
			n.RemoveChild(n, textNodes[i])
		}

		return ast.WalkContinue, nil
	})
}
