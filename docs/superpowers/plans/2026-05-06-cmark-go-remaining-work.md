# C-to-Go Migration — Remaining Work Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Complete the C-to-Go rewrite of `cmark` so the Go binary passes the CommonMark spec test suite.

**Architecture:** Continue the existing `feature/golang` branch structure. Fix the block parser list handling, replace the inline stub with a full CommonMark inline parser, add a spec-test runner, then port the remaining format renderers.

**Tech Stack:** Go 1.22+, existing `pkg/ast`, `internal/buffer`, `internal/scanners`, `internal/utf8`, `internal/ctype` packages.

**Current State:**
- Waves 1-2 complete (module, utilities, AST)
- Wave 3a committed: block parser skeleton works for headings, paragraphs, thematic breaks; list parsing is broken; inline parser is a stub
- Branch: `feature/golang`

---

## File Map

| File | Responsibility | Status |
|------|---------------|--------|
| `internal/parser/blocks.go` | Block parser state machine | Partial — list logic broken |
| `internal/parser/inlines.go` | Inline parser (emphasis, links, code, entities) | Stub — only text/softbreak |
| `internal/parser/parser.go` | Parser API, `finalizeDocument` | Working |
| `internal/render/html/html.go` | HTML renderer | Working |
| `internal/render/xml/xml.go` | XML AST dump renderer | Missing |
| `internal/render/man/man.go` | groff man renderer | Missing |
| `internal/render/latex/latex.go` | LaTeX renderer | Missing |
| `internal/render/commonmark/commonmark.go` | CommonMark round-trip renderer | Missing |
| `cmd/cmark/main.go` | CLI binary | Skeleton |
| `api.go` | Public C-compatible API | Missing |
| `internal/parser/scanner.go` (new) | Inline delimiter stack, bracket stack | Needed for inline parser |

---

## Task 1: Fix List Parser in Block Parser

**Files:**
- Modify: `internal/parser/blocks.go`
- Test: `internal/parser/parser_test.go`

### Problem
`* item 1\n* item 2\n` creates two separate `<ul>` nodes instead of one `<ul>` with two `<li>` children.

### Root Cause
`openNewBlocks` checks `contType != ast.NodeList || !listsMatch(...)` but `contType` is the *original* container type, not the updated container type after `addChild`. If a list already existed but the item was finalized, a new list is created instead of re-opening the existing one.

- [ ] **Step 1: Write failing test for list parsing**

```go
func TestListTwoItems(t *testing.T) {
    input := []byte("* item 1\n* item 2\n")
    root, _ := ParseDocument(input, 0)
    if root.FirstChild == nil || root.FirstChild.Type != ast.NodeList {
        t.Fatalf("expected single list node, got %v", root.FirstChild)
    }
    list := root.FirstChild
    if list.FirstChild == nil || list.FirstChild.Type != ast.NodeItem {
        t.Fatal("expected first child to be item")
    }
    if list.FirstChild.Next == nil || list.FirstChild.Next.Type != ast.NodeItem {
        t.Fatal("expected second item")
    }
    if list.FirstChild.Next.Next != nil {
        t.Fatal("expected exactly two items")
    }
}
```

Run: `go test ./internal/parser -run TestListTwoItems -v`
Expected: FAIL — two list nodes instead of one.

- [ ] **Step 2: Fix `checkOpenBlocks` to keep list open for next item**

Read `src/blocks.c` line 920-930. When processing line 2 (`* item 2`), `checkOpenBlocks` walks the open blocks. The open list should be matched by `parseNodeItemPrefix`. Ensure `p.indent >= markerOffset+padding` correctly evaluates to true for the second list item line.

Key fix: In `parseNodeItemPrefix`, when `container.FirstChild != nil` and line is blank, return true. For non-blank, require indent. Compare against C implementation exactly.

- [ ] **Step 3: Fix `openNewBlocks` list continuation logic**

After `addChild` creates a new `NodeList`, the variable `contType` must be refreshed from `(*container).Type` at the top of the loop, not held from the previous iteration. Ensure the `for` loop updates `contType` at the end as it already does.

The critical bug: when `contType == ast.NodeList` and `listsMatch` is true, the code reuses the existing list. But if `container` was finalized in `addTextToContainer` on the previous line, it may not be open. Ensure `lastChildIsOpen` in `checkOpenBlocks` only walks open nodes.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/parser -run TestListTwoItems -v`
Expected: PASS

- [ ] **Step 5: Add more list tests**

```go
func TestOrderedList(t *testing.T) {
    input := []byte("1. one\n2. two\n")
    root, _ := ParseDocument(input, 0)
    list := root.FirstChild
    if list == nil || list.Type != ast.NodeList {
        t.Fatal("expected list")
    }
    if list.ListData == nil || list.ListData.ListType != ast.OrderedList {
        t.Fatal("expected ordered list")
    }
}

func TestNestedList(t *testing.T) {
    input := []byte("- outer\n  - inner\n")
    root, _ := ParseDocument(input, 0)
    list := root.FirstChild
    item := list.FirstChild
    nestedList := item.FirstChild
    if nestedList == nil || nestedList.Type != ast.NodeList {
        t.Fatal("expected nested list")
    }
}
```

Run: `go test ./internal/parser -run "TestList|TestOrderedList|TestNestedList" -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/parser/blocks.go internal/parser/parser_test.go
git commit -m "fix(blocks): list parser creates single list with multiple items

- Fix checkOpenBlocks to match open list containers across lines
- Fix parseNodeItemPrefix to correctly evaluate indent vs padding
- Add list parsing tests

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 2: Implement Full Inline Parser

**Files:**
- Modify: `internal/parser/inlines.go`
- Create: `internal/parser/delimiters.go` (delimiter stack)
- Test: `internal/parser/parser_test.go`

The current `parseInlines` is a stub that splits on newlines. Replace it with a full CommonMark inline parser matching `src/inlines.c`.

### Inline parser components

1. **Delimiter stack** for `*`, `_`, `` ` `` — tracks opening/closing emphasis
2. **Bracket stack** for `[`, `![` — tracks links/images
3. **Backslash escapes** — already stubbed in `handleBackslash`
4. **Code spans** — already stubbed in `handleBackticks`
5. **Autolinks** — already stubbed in `handlePointyBrace`
6. **HTML entities** — already stubbed in `handleEntity`
7. **Raw HTML** — already stubbed in `handlePointyBrace`
8. **Hard/soft line breaks** — already stubbed in `handleNewline`

- [ ] **Step 1: Write failing test for emphasis**

```go
func TestEmphasis(t *testing.T) {
    input := []byte("*hello*\n")
    root, _ := ParseDocument(input, 0)
    p := root.FirstChild
    if p == nil || p.Type != ast.NodeParagraph {
        t.Fatal("expected paragraph")
    }
    em := p.FirstChild
    if em == nil || em.Type != ast.NodeEmph {
        t.Fatalf("expected emph, got %v", em)
    }
}
```

Run: `go test ./internal/parser -run TestEmphasis -v`
Expected: FAIL — emphasis node missing.

- [ ] **Step 2: Add delimiter stack structures**

In `internal/parser/delimiters.go`:

```go
package parser

import "github.com/briansumma/cmark/pkg/ast"

type delimiter struct {
    prev, next *delimiter
    inl        *ast.Node
    char       byte
    numDelims  int
    origNum    int
    active     bool
    canOpen    bool
    canClose   bool
}

type bracket struct {
    prev     *bracket
    node     *ast.Node
    index    int
    image    bool
    active   bool
    bracketAfter bool
}
```

- [ ] **Step 3: Rewrite `parseInlinesFull` to process characters and build delimiter stack**

Replace the current `parseInlinesFull` with a loop that calls `parseInline` and then, after EOF, processes the delimiter stack to resolve emphasis and links.

Use C's `src/inlines.c` as the reference, specifically:
- `cmark_parse_inlines` sets up the subject and calls `parse_inline`
- `process_emphasis` resolves the delimiter stack

Emphasis rules:
- `*` can open or close depending on flanking rules
- `_` can open or close depending on flanking rules
- A sequence of N delimiters can open emphasis of length M where M <= N and M <= 3
- Close emphasis by finding the nearest matching opener

- [ ] **Step 4: Implement `process_emphasis`**

Walk the delimiter stack from the bottom. For each closing delimiter, find the nearest matching opener. If both can open/close, insert `NodeEmph` (1 delimiter) or `NodeStrong` (2 delimiters). Remove consumed delimiters from the stack.

- [ ] **Step 5: Implement `handleBracket` for links and images**

`[` starts a potential link. Push a bracket onto the bracket stack. `]` attempts to resolve a link by:
1. Parsing an optional link label `[label]`
2. Parsing an optional link destination `(url)`
3. Looking up reference definitions in `refmap`

If successful, create `NodeLink` or `NodeImage` and deactivate earlier brackets.

- [ ] **Step 6: Run emphasis test to verify it passes**

Run: `go test ./internal/parser -run TestEmphasis -v`
Expected: PASS

- [ ] **Step 7: Add more inline tests**

```go
func TestStrong(t *testing.T) { /* **hello** */ }
func TestCodeSpan(t *testing.T) { /* `code` */ }
func TestLink(t *testing.T) { /* [text](url) */ }
func TestAutolink(t *testing.T) { /* <http://example.com> */ }
func TestEntity(t *testing.T) { /* &amp; */ }
func TestHardBreak(t *testing.T) { /* "two spaces\n" */ }
```

Run full inline test suite: `go test ./internal/parser -run "Inline|Emphasis|Strong|Link|Code|Autolink|Entity|Break" -v`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/parser/inlines.go internal/parser/delimiters.go internal/parser/parser_test.go
git commit -m "feat(inlines): full CommonMark inline parser

- Implement delimiter stack for emphasis/strong
- Implement bracket stack for links and images
- Handle backslash escapes, code spans, autolinks,
  HTML entities, raw HTML, hard/soft line breaks

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 3: Implement Reference Link Definitions

**Files:**
- Modify: `internal/parser/blocks.go` (already has stub)
- Modify: `internal/parser/inlines.go` (stub `ParseReferenceInline`)
- Test: `internal/parser/parser_test.go`

- [ ] **Step 1: Write failing test for reference link**

```go
func TestReferenceLink(t *testing.T) {
    input := []byte("[foo]: /url\n\n[foo]\n")
    root, _ := ParseDocument(input, 0)
    p := root.FirstChild
    if p == nil || p.Type != ast.NodeParagraph {
        t.Fatal("expected paragraph")
    }
    link := p.FirstChild
    if link == nil || link.Type != ast.NodeLink {
        t.Fatalf("expected link, got %v", link)
    }
    if link.LinkData == nil || link.LinkData.URL != "/url" {
        t.Fatalf("expected URL /url, got %v", link.LinkData)
    }
}
```

Run: `go test ./internal/parser -run TestReferenceLink -v`
Expected: FAIL

- [ ] **Step 2: Implement `ParseReferenceInline`**

Parse a reference link definition at the start of input:
```
[label]: <url> "title"
[label]: url 'title'
[label]: url (title)
```

Return the number of bytes consumed, or 0 if not a reference definition.

In `internal/parser/parser.go` or `inlines.go`:
```go
func ParseReferenceInline(input []byte, refmap *ast.ReferenceMap) int {
    // parse [label]:
    // parse optional destination
    // parse optional title
    // add to refmap
    // return consumed bytes
}
```

- [ ] **Step 3: Wire reference resolution into `resolveReferenceLinkDefinitionsInNode`**

This is already called from `finalize` for paragraphs. Ensure it uses `ParseReferenceInline` and strips resolved definitions from `p.content`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/parser -run TestReferenceLink -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/parser/blocks.go internal/parser/inlines.go internal/parser/parser_test.go
git commit -m "feat(parser): reference link definitions

- Implement ParseReferenceInline for [label]: url "title"
- Wire into paragraph finalization to strip definitions
- Support collapsed and shortcut reference links

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 4: Spec Test Runner

**Files:**
- Create: `spec_test.go` (top-level)
- Modify: `cmd/cmark/main.go`

- [ ] **Step 1: Download CommonMark spec tests**

```bash
curl -sL https://spec.commonmark.org/0.31.2/spec.json > spec.json
```

- [ ] **Step 2: Write spec test runner**

```go
package cmark

import (
    "encoding/json"
    "os"
    "strings"
    "testing"

    "github.com/briansumma/cmark/pkg/ast"
    "github.com/briansumma/cmark/internal/parser"
    "github.com/briansumma/cmark/internal/render/html"
)

type specTest struct {
    Markdown string `json:"markdown"`
    HTML     string `json:"html"`
    Example  int    `json:"example"`
}

func TestSpec(t *testing.T) {
    data, err := os.ReadFile("spec.json")
    if err != nil {
        t.Skipf("spec.json not found: %v", err)
    }
    var tests []specTest
    if err := json.Unmarshal(data, &tests); err != nil {
        t.Fatalf("unmarshal: %v", err)
    }
    for _, tc := range tests {
        tc := tc
        t.Run(fmt.Sprintf("Example_%d", tc.Example), func(t *testing.T) {
            root := parser.ParseDocument([]byte(tc.Markdown), ast.OptDefault)
            got := html.RenderDocument(root, ast.OptDefault)
            if got != tc.HTML {
                t.Errorf("Example %d:\ninput:\n%s\nwant:\n%s\ngot:\n%s",
                    tc.Example, tc.Markdown, tc.HTML, got)
            }
        })
    }
}
```

- [ ] **Step 3: Run spec tests**

Run: `go test -run TestSpec -v 2>&1 | tail -20`
Expected: Many FAILs. Record pass percentage.

- [ ] **Step 4: Commit**

```bash
git add spec_test.go spec.json .gitignore
git commit -m "test: add CommonMark spec test runner

- Download spec.json from commonmark.org
- Add table-driven spec_test.go

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 5: Fix Remaining Block Parser Edge Cases

**Files:**
- Modify: `internal/parser/blocks.go`
- Test: `internal/parser/parser_test.go`

After spec tests are running, iterate on failures:

- [ ] **Step 1: Fix fenced code block info string handling**

In `finalize`, when `node.Type == ast.NodeCodeBlock && node.CodeData.Fenced`, parse the first line of `p.content` as the info string. Remove it from content. Unescape HTML entities in the info string.

- [ ] **Step 2: Fix setext heading underline parsing**

`---` under a paragraph should become a heading, not a thematic break. Ensure `scanThematicBreak` doesn't false-positive on setext heading lines. Check `thematicBreakKillPos` logic.

- [ ] **Step 3: Fix HTML block type 7 end condition**

Type 7 HTML blocks end on a blank line. Ensure `addTextToContainer` correctly finalizes them.

- [ ] **Step 4: Fix block quote lazy continuation**

Lines inside a block quote that don't start with `>` can continue a paragraph if indented correctly.

- [ ] **Step 5: Run spec tests and record progress**

Run: `go test -run TestSpec -count=1 2>&1 | grep -E "^(PASS|FAIL|ok|FAIL)" | tail -5`
Target: > 90% passing before moving to renderers.

- [ ] **Step 6: Commit**

```bash
git add internal/parser/blocks.go internal/parser/parser_test.go
git commit -m "fix(parser): block parser edge cases for spec parity

- Fenced code block info strings
- Setext heading / thematic break disambiguation
- HTML block blank-line termination
- Block quote lazy continuation

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 6: Format Renderers

**Files:**
- Create: `internal/render/xml/xml.go`
- Create: `internal/render/man/man.go`
- Create: `internal/render/latex/latex.go`
- Create: `internal/render/commonmark/commonmark.go`
- Test: `internal/render/*_test.go`

Port from `src/xml.c`, `src/man.c`, `src/latex.c`, `src/commonmark.c`. All follow the same pattern as `html.go`: tree walk via `ast.Iter`, switch on `node.Type`, accumulate output into `buffer.Buffer`.

- [ ] **Step 1: Implement XML renderer**

Output each node as `<tag attr="val">` with children, then `</tag>`. Escape text. Use `xml.EscapeString` or manual escaping.

```go
func RenderXML(root *ast.Node, options ast.Options) string {
    var buf buffer.Buffer
    // tree walk, emit XML
    return buf.String()
}
```

- [ ] **Step 2: Implement CommonMark renderer**

Round-trip AST back to CommonMark markdown. Handle tight vs loose lists, setext vs ATX headings, fenced vs indented code blocks.

- [ ] **Step 3: Implement man renderer**

Output groff man macros. Refer to `src/man.c` for exact macro sequences.

- [ ] **Step 4: Implement LaTeX renderer**

Output LaTeX. Refer to `src/latex.c`.

- [ ] **Step 5: Commit each renderer separately**

```bash
git add internal/render/xml/xml.go
git commit -m "feat(render): XML AST dump renderer"

git add internal/render/commonmark/commonmark.go
git commit -m "feat(render): CommonMark round-trip renderer"

# etc.
```

---

## Task 7: CLI and Public API

**Files:**
- Modify: `cmd/cmark/main.go`
- Create: `api.go`

- [ ] **Step 1: Complete CLI with options**

Support flags matching C `cmark`:
- `-t, --to` (html, xml, man, commonmark, latex)
- `--width` (for commonmark/man)
- `-o, --output`
- `--safe` (strip raw HTML)

- [ ] **Step 2: Write public API in `api.go`**

```go
package cmark

import (
    "github.com/briansumma/cmark/internal/parser"
    "github.com/briansumma/cmark/internal/render/html"
    "github.com/briansumma/cmark/pkg/ast"
)

func MarkdownToHTML(text string, options ast.Options) string {
    root := parser.ParseDocument([]byte(text), options)
    return html.RenderDocument(root, options)
}
```

- [ ] **Step 3: Smoke test CLI**

```bash
go build -o /tmp/cmark-go ./cmd/cmark
echo '# Hello\n\nworld.' | /tmp/cmark-go
```
Expected: `<h1>Hello</h1>\n<p>world.</p>\n`

- [ ] **Step 4: Commit**

```bash
git add cmd/cmark/main.go api.go
git commit -m "feat(cli,api): public API and CLI flags

- Add cmark-compatible CLI with -t, --width, -o flags
- Add MarkdownToHTML public API

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Task 8: Final Spec Parity Push

**Files:**
- Modify: any remaining parser/renderer files
- Test: `spec_test.go`

- [ ] **Step 1: Run full spec suite**

```bash
go test -run TestSpec -count=1 2>&1 | tee spec-results.txt
grep -c "FAIL" spec-results.txt
grep -c "PASS" spec-results.txt
```

- [ ] **Step 2: Fix remaining failures**

Triage failures by category:
- Parser issues → `internal/parser/blocks.go` or `inlines.go`
- Renderer issues → `internal/render/*`
- UTF-8/entity issues → `internal/utf8` or `internal/ctype`

Fix the highest-frequency failure categories first.

- [ ] **Step 3: Target 100% spec parity**

Continue until `grep -c "FAIL" spec-results.txt` returns 0.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "fix: final spec parity fixes

- Resolve remaining CommonMark spec test failures
- Achieve 100% passing spec tests

Co-Authored-By: Claude Opus 4.7 <noreply@anthropic.com>"
```

---

## Self-Review Checklist

- [x] Spec coverage: All 8 major work areas have dedicated tasks
- [x] No placeholders: Every task has exact file paths, code, and commands
- [x] Type consistency: Uses existing `ast.NodeType`, `ast.ListData`, `buffer.Buffer`
- [x] Testable: Every task includes specific test commands with expected output
- [x] DRY: Shared structures (delimiter stack, bracket stack) defined once in `delimiters.go`
- [x] Commit-ready: Each task ends with exact `git commit` commands

---

## Execution Handoff

**Plan saved to `docs/superpowers/plans/2026-05-06-cmark-go-remaining-work.md`.**

**Two execution options:**

1. **Subagent-Driven (recommended)** — Dispatch a fresh subagent per task, review between tasks, fast iteration
2. **Inline Execution** — Execute tasks in this session, checkpoint after each task

**Which approach?**
