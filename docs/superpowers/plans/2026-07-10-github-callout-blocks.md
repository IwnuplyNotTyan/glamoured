# GitHub-Style Callout Blocks Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Render `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]` as styled callout blocks inside blockquotes.

**Architecture:** Single-file change in `ansi/elements.go`. The text handler (case `ast.KindText`) detects callout markers by walking up the AST to check if inside a blockquote. When found, it replaces the marker text with a styled label and colors the blockquote's `│` indent prefix by modifying the `IndentToken` on the block stack.

**Tech Stack:** Go, goldmark AST, `strings`, `golang.org/x/net/html` (already a dep)

---

### Task 1: Add callout detection and styled label rendering to text handler

**Files:**
- Modify: `ansi/elements.go` (case `ast.KindText`, lines 173-185)
- Test: `ansi/renderer_test.go`

- [ ] **Step 1: Read current text handler**

Read `ansi/elements.go` lines 172-186 to see the current KindText handler.

- [ ] **Step 2: Add callout constants and helper function**

Add before `func (tr *ANSIRenderer) NewElement` (around line 43):

```go
type calloutType int

const (
	calloutNone calloutType = iota
	calloutNote
	calloutTip
	calloutImportant
	calloutWarning
	calloutCaution
)

var calloutPatterns = map[string]struct {
	typ   calloutType
	label string
	color string
	icon  string
}{
	"note":      {calloutNote, "Note:", "39", "ℹ"},
	"tip":       {calloutTip, "Tip:", "42", "💡"},
	"important": {calloutImportant, "Important:", "129", "❗"},
	"warning":   {calloutWarning, "Warning:", "214", "⚠"},
	"caution":   {calloutCaution, "Caution:", "196", "🛑"},
}

func detectCallout(text string) (calloutType, string, bool) {
	for prefix, info := range calloutPatterns {
		marker := "[!" + prefix + "]"
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(text)), marker) {
			return info.typ, info.label, true
		}
		// Also check uppercase variant directly
		upperMarker := "[!" + strings.ToUpper(prefix) + "]"
		if strings.HasPrefix(text, upperMarker) {
			return info.typ, info.label, true
		}
	}
	return calloutNone, "", false
}
```

Note: Use `strings.ToUpper` on the prefix so the marker is always checked as `[!NOTE]`, `[!WARNING]` etc.

- [ ] **Step 3: Update the text handler to detect callouts**

Replace lines 173-185:

```go
case ast.KindText:
    n := node.(*ast.Text)
    s := string(n.Segment.Value(source))

    // Check if this text is inside a blockquote paragraph
    if typ, label, ok := isCalloutText(s); ok {
        info := calloutPatterns[strings.ToLower(label)]
        parentBq := findParentBlockquote(n)
        if parentBq != nil {
            // Color the blockquote's indent token
            colorBqIndent(parentBq, info.color)
            // Render styled label instead of marker text
            return calloutLabelElement(info.icon, info.label, info.color)
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
```

- [ ] **Step 4: Add helper functions for callout detection**

Add after the `detectCallout` function:

```go
func isCalloutText(text string) (calloutType, string, bool) {
	trimmed := strings.TrimSpace(text)
	for _, info := range calloutPatterns {
		marker := "[!" + info.label + "]"
		// Actually, we need the lowercased prefix key
	}
	// Simplified:
	for key, info := range calloutPatterns {
		marker := "[!" + key + "]"
		if strings.HasPrefix(strings.ToLower(trimmed), marker) {
			return info.typ, key, true
		}
		markerUpper := "[!" + strings.ToUpper(key) + "]"
		if strings.HasPrefix(trimmed, markerUpper) {
			return info.typ, key, true
		}
	}
	return calloutNone, "", false
}

func findParentBlockquote(n ast.Node) ast.Node {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == ast.KindBlockquote {
			return p
		}
	}
	return nil
}

func colorBqIndent(bqNode ast.Node, color string) {
	// We'll use the ANSIColor int for the color
	token := fmt.Sprintf("\x1b[38;5;%sm│ \x1b[0m", color)
	// Store this on the context or blockstack — to be implemented
}

func calloutLabelElement(icon, label, color string) Element {
	return Element{
		Renderer: &BaseElement{
			Token: fmt.Sprintf("%s %s ", icon, label),
			Style: StylePrimitive{
				Color: &color,
				Bold:  boolPtr(true),
			},
		},
	}
}
```

Wait, this is getting complex. Let me simplify the approach. The helper functions need to interact with the block stack, not the AST directly. Let me reorganize.

- [ ] **Step 5: Refined approach — modify ctx.blockStack in text handler**

The text handler has access to `ctx` (RenderContext). We can modify the block stack directly. Here's the actual implementation:

```go
case ast.KindText:
    n := node.(*ast.Text)
    s := string(n.Segment.Value(source))

    // Check for callout markers inside blockquotes
    if typ, key, ok := detectCallout(s); ok && typ != calloutNone {
        if info, found := calloutPatterns[key]; found {
            if isInsideBlockquote(n) {
                // Color the blockquote's indent token on the stack
                colorBlockquoteIndent(ctx, info.color)
                // Render the styled callout label
                return Element{
                    Renderer: &BaseElement{
                        Token: info.icon + " " + info.label + " ",
                        Style: StylePrimitive{
                            Color: &info.color,
                            Bold:  boolPtr(true),
                        },
                    },
                }
            }
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
```

- [ ] **Step 6: Add helper functions**

Add after the `parseHTMLImage` function at the end of elements.go:

```go
func isInsideBlockquote(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if p.Kind() == ast.KindBlockquote {
			return true
		}
	}
	return false
}

func colorBlockquoteIndent(ctx RenderContext, color string) {
	bs := ctx.blockStack
	// Walk the stack to find the blockquote (BlockElement with BlockQuote-style indent)
	// The blockquote is typically the parent of the paragraph
	for i := len(*bs) - 1; i >= 0; i-- {
		e := &(*bs)[i]
		if e.Style.Indent != nil && e.Style.IndentToken != nil {
			styledToken := fmt.Sprintf("\x1b[38;5;%sm%s\x1b[0m", color, *e.Style.IndentToken)
			e.Style.IndentToken = &styledToken
			return
		}
	}
}
```

Wait, this heuristic (looking for an element with Indent and IndentToken set) would match any indented block element, not just blockquotes. Let me use a different approach.

Actually, the simplest approach: we know the callout is inside a blockquote, and the blockquote's BlockElement is somewhere on the stack with an IndentToken of `"│ "`. But how do we identify it specifically?

Let me add a field to `BlockElement`:

```go
type BlockElement struct {
    Block       *bytes.Buffer
    Style       StyleBlock
    Margin      bool
    Newline     bool
    IsBlockquote bool  // set by blockquote handler
}
```

Then in the blockquote handler:
```go
case ast.KindBlockquote:
    e := &BlockElement{
        Block:        &bytes.Buffer{},
        Style:        cascadeStyle(ctx.blockStack.Current().Style, ctx.options.Styles.BlockQuote, false),
        Margin:       true,
        IsBlockquote: true,
    }
```

And in the color function:
```go
func colorBlockquoteIndent(ctx RenderContext, color string) {
    bs := ctx.blockStack
    for i := len(*bs) - 1; i >= 0; i-- {
        if (*bs)[i].IsBlockquote {
            styledToken := fmt.Sprintf("\x1b[38;5;%sm%s\x1b[0m", color, *(*bs)[i].Style.IndentToken)
            (*bs)[i].Style.IndentToken = &styledToken
            return
        }
    }
}
```

This is clean and doesn't rely on heuristics.

- [ ] **Step 7: Add the `IsBlockquote` field to BlockElement**

In `ansi/blockelement.go`, modify the struct:

```go
type BlockElement struct {
	Block        *bytes.Buffer
	Style        StyleBlock
	Margin       bool
	Newline      bool
	IsBlockquote bool
}
```

- [ ] **Step 8: Set `IsBlockquote` in the blockquote handler**

In `ansi/elements.go`, update the blockquote case:

```go
case ast.KindBlockquote:
    e := &BlockElement{
        Block:        &bytes.Buffer{},
        Style:        cascadeStyle(ctx.blockStack.Current().Style, ctx.options.Styles.BlockQuote, false),
        Margin:       true,
        IsBlockquote: true,
    }
```

Wait, actually looking at the current code, it doesn't use `&`. Let me re-read:

```go
case ast.KindBlockquote:
    e := &BlockElement{
        Block:  &bytes.Buffer{},
        Style:  cascadeStyle(ctx.blockStack.Current().Style, ctx.options.Styles.BlockQuote, false),
        Margin: true,
    }
    return Element{
        Entering: "\n",
        Renderer: e,
        Finisher: e,
    }
```

OK so it already takes a pointer to BlockElement. I just need to add `IsBlockquote: true`.

- [ ] **Step 9: Remove unused helper and finalize**

Actually, I had `detectCallout` and then `isCalloutText`. Let me consolidate. The `detectCallout` function returns `(calloutType, string, bool)` where the second return is the key (like "note", "tip"). Actually, I can just return the calloutPatterns struct directly or use a simpler approach.

Let me use a simpler function:

```go
func detectCallout(text string) (color string, label string, icon string, ok bool) {
    trimmed := strings.TrimSpace(text)
    for _, info := range calloutPatterns {
        marker := "[!" + info.key + "]"
        if strings.HasPrefix(strings.ToLower(trimmed), marker) {
            return info.color, info.label, info.icon, true
        }
    }
    return "", "", "", false
}
```

Wait, I need to define `calloutPatterns` differently. Let me use a cleaner structure:

```go
type calloutInfo struct {
    label string
    color string
    icon  string
}

var calloutPatterns = map[string]calloutInfo{
    "note":      {"Note:", "39", "ℹ"},
    "tip":       {"Tip:", "42", "💡"},
    "important": {"Important:", "129", "❗"},
    "warning":   {"Warning:", "214", "⚠"},
    "caution":   {"Caution:", "196", "🛑"},
}
```

And `detectCallout`:

```go
func detectCallout(text string) (calloutInfo, bool) {
    trimmed := strings.TrimSpace(text)
    for key, info := range calloutPatterns {
        marker := "[!" + key + "]"
        if strings.HasPrefix(strings.ToLower(trimmed), marker) {
            return info, true
        }
    }
    return calloutInfo{}, false
}
```

This is cleaner.

- [ ] **Step 10: Add `"fmt"` to imports if missing**

Check that `"fmt"` is imported in `ansi/elements.go`. It's already imported.

Add `"strings"` to imports — check if already there. It is.

- [ ] **Step 11: Build and test**

Run:
```bash
cd /home/q/files/prjkt/glamoured && go build ./... && go test ./...
```

Expected: all tests pass.

---

### Task 2: Write tests for callout blocks

**Files:**
- Modify: `ansi/elements.go` (already done)
- Test: `ansi/renderer_test.go` or `ansi/elements_test.go`

Check if there's an existing renderer test file.

- [ ] **Step 1: Check for existing test file**

Run:
```bash
ls /home/q/files/prjkt/glamoured/ansi/*test*.go
```

- [ ] **Step 2: Add a callout test function**

In the test file, add:

```go
func TestCalloutBlocks(t *testing.T) {
    tests := []struct {
        name     string
        markdown string
    }{
        {"note", "> [!NOTE]\n> This is a note\n"},
        {"tip", "> [!TIP]\n> This is a tip\n"},
        {"important", "> [!IMPORTANT]\n> This is important\n"},
        {"warning", "> [!WARNING]\n> This is a warning\n"},
        {"caution", "> [!CAUTION]\n> This is a caution\n"},
        {"case insensitive", "> [!note]\n> lower case\n"},
        {"with content after", "> [!NOTE] Content on same line\n"},
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

            out, err := r.RenderBytes([]byte(tt.markdown))
            if err != nil {
                t.Fatal(err)
            }

            clean := stripANSICodes(string(out))
            if !strings.Contains(clean, "│") {
                t.Error("expected blockquote prefix '│' in output")
            }
            if strings.Contains(clean, "[!") {
                t.Error("callout marker still present in output")
            }
        })
    }
}

func stripANSICodes(s string) string {
    var b strings.Builder
    inEsc := false
    for _, c := range s {
        if inEsc {
            if c == 'm' {
                inEsc = false
            }
            continue
        }
        if c == '\x1b' {
            inEsc = true
            continue
        }
        b.WriteRune(c)
    }
    return b.String()
}
```

- [ ] **Step 3: Run the tests**

Run:
```bash
cd /home/q/files/prjkt/glamoured && go test ./ansi/... -run TestCalloutBlocks -v
```

Expected: all tests pass.

---

### Task 3: Manual verification

- [ ] **Step 1: Write a manual test program**

```go
package main

import (
    "fmt"
    "strings"
    "charm.land/glamour/v2"
)

func stripANSI(s string) string {
    var b strings.Builder
    inEsc := false
    for _, c := range s {
        if inEsc {
            if c == 'm' { inEsc = false }
            continue
        }
        if c == '\x1b' { inEsc = true; continue }
        b.WriteRune(c)
    }
    return b.String()
}

func main() {
    r, _ := glamour.NewTermRenderer(
        glamour.WithWordWrap(80),
        glamour.WithStandardStyle("dark"),
    )

    md := "# Callout Test\n\n> [!NOTE]\n> Useful information here\n\n> [!WARNING]\n> Be careful!\n\nRegular text.\n"
    out, _ := r.RenderBytes([]byte(md))
    clean := stripANSI(string(out))
    fmt.Println(clean)
}
```

- [ ] **Step 2: Run and verify**

Run:
```bash
cd /tmp/opencode && mkdir -p callouttest && cat > callouttest/main.go << 'EOF'
... (test code above)
EOF
cd /tmp/opencode/callouttest && go mod init callouttest && go mod edit -replace charm.land/glamour/v2=/home/q/files/prjkt/glamoured && go mod tidy && go run main.go
```

Expected: callout labels appear styled (Note:, Warning:) and not as raw `[!NOTE]` text.

---

### Task 4: Commit

- [ ] **Step 1: Run full test suite**

Run:
```bash
cd /home/q/files/prjkt/glamoured && go build ./... && go test ./... -count=1
```

Expected: all tests pass.

- [ ] **Step 2: Commit**

```bash
cd /home/q/files/prjkt/glamoured && git add -A && git commit -m "feat: add GitHub-style callout blocks (NOTE, TIP, IMPORTANT, WARNING, CAUTION)"
```
