# CommonMark Spec Parity — Final 37 Tests

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the remaining 37 failing CommonMark spec tests (615/652 → 652/652) in the Go cmark rewrite on `feature/golang`.

**Architecture:** The failures cluster into 4 root causes, each independent. Fix order matters: Tasks 1-3 (reference/link fixes) are independent of each other but must all land before Task 4 (list parsing), since list tests depend on correct reference and inline handling. Task 5 (tight/loose) depends on Task 4.

**Tech Stack:** Go 1.21+, `go test -run TestSpec`, `spec.json` (652 examples)

**Testing pattern:** Every task uses the spec tests as its TDD harness. Run individual examples with `go test -v -run "TestSpec/Example_N$" .` and the full suite with `go test -run TestSpec .`.

**Critical rule:** After EVERY code change, run the full 652-test suite and confirm the total passing count does not decrease. Regressions must be fixed immediately before moving on.

---

## Task 1: Fix Reference Definition Parsing (3 tests: 197, 208, 217)

**Root causes:**
- Labels reject `\n`/`\r` but should allow line continuations (Ex 208, 541)
- Titles spanning blank lines incorrectly accepted (Ex 197)
- Multi-definition sequences broken by title-on-next-line handling (Ex 217)

**Files:**
- Modify: `internal/parser/parser.go` — `ParseReferenceInline` function (lines 95-199)
- Test: `go test -v -run "TestSpec/Example_(197|208|217)$" .`

### Step 1.1: Allow newlines in reference labels

- [ ] **Read the current label parser** (parser.go lines 100-118). Note line 113 rejects `\n` and `\r`.

- [ ] **Fix: Remove `\n`/`\r` rejection from label loop.** Replace the label scanning loop. The new loop must track raw bytes (including `\n`) in the label, skip `\[` as backslash-escaped brackets, and stop at unescaped `]`. Do NOT use a `strings.Builder` — use the raw input slice `input[labelStart:labelEnd]`.

```go
// Parse label: scan past [...] to find "]:"
labelStart := 1
labelEnd := labelStart
for labelEnd < len(input) {
    if input[labelEnd] == '\\' && labelEnd+1 < len(input) && isPunct(input[labelEnd+1]) {
        labelEnd += 2
        continue
    }
    if input[labelEnd] == ']' {
        break
    }
    if input[labelEnd] == '[' {
        return 0
    }
    labelEnd++
}
```

Then use `label := string(input[labelStart:labelEnd])` instead of `labelBuilder.String()`. Remove the `labelBuilder` variable and `strings` import if unused.

- [ ] **Run:** `go test -v -run "TestSpec/Example_208$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 1.2: Reject titles that span blank lines

- [ ] **Read Ex 197:** Input `[foo]: /url 'title\n\nwith blank line'\n\n[foo]\n`. Expected: the definition is invalid because the title contains a blank line.

- [ ] **Fix:** In `ParseReferenceInline`, after calling `scanners.ScanLinkTitle`, verify the matched title does not contain a blank line (two consecutive `\n` characters, or `\r\n\r\n`). If it does, treat the title as not found:

```go
if titleLen > 0 {
    titleRaw := string(input[titleStart : titleStart+titleLen])
    if strings.Contains(titleRaw, "\n\n") || strings.Contains(titleRaw, "\r\n\r\n") {
        titleLen = 0 // reject title spanning blank line
    }
}
```

This requires re-adding `"strings"` to the import block.

- [ ] **Run:** `go test -v -run "TestSpec/Example_197$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 1.3: Fix title-on-next-line consuming too much input

- [ ] **Read Ex 217:** Three consecutive definitions, the second has its title on the next line (`[bar]: /bar-url\n  "bar"`). The issue is that `skipThroughBlank` after the URL jumps past the newline to the title, but then the end-of-definition calculation doesn't account for multi-definition sequences correctly.

- [ ] **Fix:** After computing `end` at line ~190-195, ensure `end` does not extend past a blank line. The `skipBlankLines` call should only skip a single terminating newline, not consume the start of the next definition.

Replace:
```go
end += skipBlankLines(input[end:])
```
With:
```go
// Skip at most one line ending to terminate this definition
if end < len(input) && input[end] == '\r' {
    end++
}
if end < len(input) && input[end] == '\n' {
    end++
}
```

- [ ] **Run:** `go test -v -run "TestSpec/Example_217$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 1.4: Commit

- [ ] `git add internal/parser/parser.go && git commit -m "Fix reference definition label newlines, title blank lines, and end calculation"`

---

## Task 2: Fix Link Reference Matching (7 tests: 512, 528, 540, 541, 545, 569, 571)

**Root causes:**
- Unicode case folding incomplete — `ẞ` doesn't fold to match `SS` (Ex 540)
- `normalizeReference` in `pkg/ast/ast.go` resolves backslash escapes but shouldn't — that's the inline parser's job (Ex 545)
- `handleCloseBracket` doesn't reset `subj.pos` on failed reference lookup after `linkLabel` succeeds (Ex 545, 569, 571)
- Nested unmatched brackets prevent outer link from matching (Ex 512, 528)

**Files:**
- Modify: `pkg/ast/ast.go` — `normalizeReference`, add `unicodeCaseFold` (lines 580-620)
- Modify: `internal/parser/inlines.go` — `handleCloseBracket` (lines 529-609)
- Test: `go test -v -run "TestSpec/Example_(512|528|540|541|545|569|571)$" .`

### Step 2.1: Add Unicode case folding to normalizeReference

- [ ] **Add imports** to `pkg/ast/ast.go`: `"unicode"` and `"unicode/utf8"`.

- [ ] **Add `unicodeCaseFold` function** before `isPunct`:

```go
func unicodeCaseFold(s string) string {
    var b strings.Builder
    b.Grow(len(s))
    for i := 0; i < len(s); {
        r, size := utf8.DecodeRuneInString(s[i:])
        if r < 128 {
            if r >= 'A' && r <= 'Z' {
                r += 'a' - 'A'
            }
            b.WriteByte(byte(r))
        } else {
            if expanded, ok := caseFoldExpand[r]; ok {
                b.WriteString(expanded)
            } else {
                b.WriteRune(unicode.ToLower(r))
            }
        }
        i += size
    }
    return b.String()
}

var caseFoldExpand = map[rune]string{
    0x00DF: "ss", 0x0130: "i̇", 0x0149: "ʼn",
    0x01F0: "ǰ", 0x0390: "ΐ",
    0x03B0: "ΰ", 0x0587: "եւ",
    0x1E96: "ẖ", 0x1E97: "ẗ", 0x1E98: "ẘ",
    0x1E99: "ẙ", 0x1E9A: "aʾ", 0x1E9E: "ss",
    0x1F50: "ὐ",
    0xFB00: "ff", 0xFB01: "fi", 0xFB02: "fl",
    0xFB03: "ffi", 0xFB04: "ffl", 0xFB05: "st", 0xFB06: "st",
}
```

- [ ] **Replace** `strings.ToLower(label)` with `unicodeCaseFold(label)` in `normalizeReference`.

- [ ] **Run:** `go test -v -run "TestSpec/Example_540$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 2.2: Remove backslash escape resolution from normalizeReference

- [ ] In `normalizeReference` (pkg/ast/ast.go), the whitespace collapsing loop currently resolves `\!` → `!`. Remove that. The loop should ONLY collapse whitespace:

Replace the loop body:
```go
for i := 0; i < len(label); i++ {
    c := label[i]
    if c == '\\' && i+1 < len(label) && isPunct(label[i+1]) {
        i++
        c = label[i]
        b.WriteByte(c)
        lastWasSpace = false
    } else if isSpace(c) {
```

With:
```go
for i := 0; i < len(label); i++ {
    c := label[i]
    if isSpace(c) {
```

- [ ] **Run:** `go test -v -run "TestSpec/Example_545$" .` — check if behavior changes
- [ ] **Run full suite** and confirm ≥ 615 passing (some link tests may now fail differently — that's expected, we fix them in the next step)

### Step 2.3: Fix handleCloseBracket position reset and bracket-after logic

This is the trickiest fix. The C reference's `handle_close_bracket` does three things differently:

1. `initial_pos` is set AFTER advancing past `]` (not before).
2. When `link_label` succeeds but the reference lookup fails, `subj->pos` is reset in the `noMatch:` branch (not inside the try-reference block).
3. The `bracket_after` flag on the PREVIOUS bracket is set when a `[` is pushed, preventing collapsed-reference resolution for the previous opener.

**Fix `handleCloseBracket` in `internal/parser/inlines.go`:**

```go
func handleCloseBracket(subj *subject, parent *ast.Node) *ast.Node {
    subj.pos++ // advance past ]
    initialPos := subj.pos  // C: initial_pos set AFTER advance

    opener := subj.lastBracket
    if opener == nil {
        n := makeStr(subj, subj.pos-1, subj.pos-1, "]")
        parent.AppendChild(n)
        return n
    }

    isImage := opener.image
    if !isImage && subj.noLinkOpeners {
        subj.popBracket()
        n := makeStr(subj, subj.pos-1, subj.pos-1, "]")
        parent.AppendChild(n)
        return n
    }

    afterLinkTextPos := subj.pos
    matched := false
    var urlStr, titleStr string

    // Try inline link: ( url "title" )
    if subj.peekChar() == '(' {
        sps := scanners.ScanSpaceChars(subj.input[subj.pos+1:])
        n, urlBytes := scanLinkURL(subj.input, subj.pos+1+sps)
        if n > -1 {
            endurl := subj.pos + 1 + sps + n
            starttitle := endurl + scanners.ScanSpaceChars(subj.input[endurl:])
            endtitle := starttitle
            if starttitle != endurl {
                titleLen := scanners.ScanLinkTitle(subj.input[starttitle:])
                if titleLen > 0 {
                    endtitle = starttitle + titleLen
                }
            }
            endall := endtitle + scanners.ScanSpaceChars(subj.input[endtitle:])
            if subj.peekAt(endall) == ')' {
                subj.pos = endall + 1
                urlStr = cleanURL(string(urlBytes))
                if starttitle != endurl {
                    titleStr = cleanTitle(string(subj.input[starttitle:endtitle]))
                }
                matched = true
            } else {
                subj.pos = afterLinkTextPos
            }
        }
    }

    // Try reference link
    var rawLabel string
    foundLabel := false
    if !matched {
        rawLabel, foundLabel = linkLabel(subj)
        if !foundLabel {
            // If we have a shortcut reference link, back up
            subj.pos = initialPos
        }
        if (!foundLabel || rawLabel == "") && !opener.bracketAfter {
            rawLabel = string(subj.input[opener.position : initialPos-1])
            foundLabel = true
        }
        if foundLabel {
            ref := subj.refmap.Lookup(rawLabel)
            if ref != nil {
                urlStr = ref.URL
                titleStr = ref.Title
                matched = true
            }
        }
    }

    // noMatch
    if !matched {
        subj.popBracket()
        subj.pos = initialPos  // C: reset pos on noMatch
        subj.noLinkOpeners = true
        n := makeStr(subj, subj.pos-1, subj.pos-1, "]")
        parent.AppendChild(n)
        return n
    }

    // ... rest of match handling stays the same ...
```

Key changes from current code:
1. `initialPos` set after `subj.pos++` (line 2 of function)
2. `rawLabel` extraction uses `initialPos-1` (one before current pos, which is `]` position)
3. `subj.pos = initialPos` added to noMatch branch

- [ ] Apply these changes
- [ ] **Run:** `go test -v -run "TestSpec/Example_(545|569|571)$" .` — expect improvements
- [ ] **Run full suite** and confirm ≥ 615 passing — **if regressions**, check collapsed reference tests (Ex 552, 553, 554, 555, 566, 576, 584, 585, 586). These use `[foo][]` syntax. The fix for those: when `linkLabel` returns an empty string for `[]` and the lookup fails for `""`, the code at `(!foundLabel || rawLabel == "")` should fall through to try the opener content. Verify this path works.

### Step 2.4: Fix nested brackets (Ex 512, 528)

- [ ] **Read the C reference** `handle_close_bracket` — when a `]` is found with no match, the C code does NOT set `no_link_openers` for images. It only sets the flag for non-image brackets. Also, it does NOT prevent further brackets from being pushed.

- [ ] **Fix:** In the noMatch branch, only set `noLinkOpeners` for non-image openers:
```go
if !isImage {
    subj.noLinkOpeners = true
}
```

For nested brackets like `[link [foo [bar]]]`, the inner `[bar]` fails, pops its bracket, but the outer `[link` bracket should still be on the stack. Verify by tracing the bracket stack.

- [ ] **Run:** `go test -v -run "TestSpec/Example_(512|528)$" .` — these are the hardest; may need additional bracket stack debugging
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 2.5: Commit

- [ ] `git add pkg/ast/ast.go internal/parser/inlines.go && git commit -m "Fix Unicode case folding, reference label normalization, and bracket handling"`

---

## Task 3: Fix Singleton Edge Cases (3 tests: 9, 61, 175)

**Root causes:**
- Ex 9: Tab handling in nested list continuation — tab expands to wrong column
- Ex 61: `<hr />` in list item missing preceding newline in renderer
- Ex 175: HTML block type 6 inside list item shouldn't continue past the `<li>` boundary

**Files:**
- Modify: `internal/render/html/html.go` — thematic break rendering in list items
- Modify: `internal/parser/blocks.go` — tab handling, HTML block in list item
- Test: `go test -v -run "TestSpec/Example_(9|61|175)$" .`

### Step 3.1: Fix thematic break rendering in list items (Ex 61)

- [ ] **Read the actual vs expected output:**
  - Got: `<li><hr />\n</li>` (no newline before `<hr />`)
  - Want: `<li>\n<hr />\n</li>` (newline before `<hr />`)

- [ ] **Fix:** In `internal/render/html/html.go`, when rendering `NodeThematicBreak`, it currently writes `<hr />\n`. The issue is that the `<li>` rendering doesn't add a newline before child block elements. Check the list item renderer — it should emit `<li>\n` when the item contains block-level children (not just inline text).

Look at the C renderer's `S_render_node` for `CMARK_NODE_ITEM` — it uses `cmark_html_render_cr(html)` which outputs a newline if not already at start of line.

In the Go renderer, add a newline after `<li>` when the first child is a block element:

```go
case ast.NodeItem:
    b.WriteString("<li>")
    if node.FirstChild != nil && node.FirstChild.IsBlock() {
        b.WriteString("\n")
    }
```

- [ ] **Run:** `go test -v -run "TestSpec/Example_61$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 3.2: Fix tab expansion in list continuation (Ex 9)

- [ ] **Read Ex 9:** Input ` - foo\n   - bar\n\t - baz`. The tab before ` - baz` should expand to 4 spaces (since column is 0, tab stop at 4), giving indent of 4 + 1 = 5, which is enough for a sublist continuation.

- [ ] **Debug** by adding a temporary print in `parseNodeItemPrefix` to log `p.indent`, `container.ListData.MarkerOffset`, and `container.ListData.Padding` for each item.

- [ ] **Fix** based on findings — likely related to how `advanceOffset` handles tabs when `columns=true` vs `columns=false`.

- [ ] **Run:** `go test -v -run "TestSpec/Example_9$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 3.3: Fix HTML block type 6 in list item (Ex 175)

- [ ] **Read Ex 175:** Input `- <div>\n- foo`. The `<div>` should start an HTML block (type 6) inside the first list item, but it should NOT continue into the second list item.

- [ ] **Debug:** Check if the issue is that the HTML block (type 6) prevents the second `- foo` from being parsed as a new list item. HTML blocks type 6 end on a blank line — but a new list item marker should also close them.

- [ ] **Fix** based on findings.

- [ ] **Run:** `go test -v -run "TestSpec/Example_175$" .` — expect PASS
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 3.4: Commit

- [ ] `git add internal/parser/blocks.go internal/render/html/html.go && git commit -m "Fix tab expansion, thematic break rendering, and HTML block in list items"`

---

## Task 4: Fix List Item Creation and Continuation (24 tests)

**This is the hardest task.** The 24 list failures (12 "List items" + 12 "Lists") share interconnected root causes in the block parser.

**Root causes identified from comparing got vs want:**

| Pattern | Examples | Root Cause |
|---------|----------|------------|
| Sublist not created | 294, 296, 298, 299, 307, 319, 323, 325, 326 | `addChild` for new list item not descending into existing items correctly |
| Loose when should be tight | 255, 257, 260, 276, 278 | Items with only 1 space of content after marker treated as continuation |
| `<p>` wrapping wrong | 308, 315, 317, 318, 320, 321 | Tight/loose detection wrong for blank-line-separated items |
| Lazy continuation wrong | 292, 293, 303 | Paragraph continuation across blockquote/list boundaries |

**Files:**
- Modify: `internal/parser/blocks.go` — `openNewBlocks`, `addTextToContainer`, `parseNodeItemPrefix`, `finalize`
- Test: `go test -v -run "TestSpec/Example_(255|257|260|276|278|292|293|294|296|298|299|300|303|307|308|315|317|318|319|320|321|323|325|326)$" .`

### Step 4.1: Understand the current behavior

- [ ] **Run each of the 24 failing tests** and categorize the actual output vs expected. Group by error pattern.

- [ ] **Add temporary debug logging** to `openNewBlocks` (at the list marker detection branch, ~line 481) to print:
  - Current container type and depth
  - `p.indent`, `p.firstNonspace`, `listMatch`
  - Whether `listsMatch` returns true/false

### Step 4.2: Fix sublist creation

The most common pattern: sublists aren't being created. Compare Go vs C carefully:

- [ ] **Check `canContainType`** — make sure `NodeItem` can contain `NodeList` (for sublists).

- [ ] **Check `addChild`** — the C code's `add_child` walks up the tree closing open blocks until it finds one that can contain the new node. This is what allows a new list marker inside a list item to create a sublist. Verify the Go `addChild` does the same.

- [ ] **Check the list marker condition** at line 481:
```go
} else if (!indented || contType == ast.NodeList) && p.indent < 4 && listMatch > 0 {
```
In C, the condition is:
```c
(!indented || cont_type == CMARK_NODE_LIST) && parser->indent < 4
```
These should match. But the `contType` variable is updated as we iterate — make sure it's being updated after sublists are created.

- [ ] Apply fixes based on findings
- [ ] **Run:** `go test -v -run "TestSpec/Example_294$" .` — the simplest sublist case
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 4.3: Fix padding and continuation

- [ ] **Check `parseNodeItemPrefix`** — compare with C's `parse_node_item_prefix`. The padding calculation determines how many spaces of indent are needed for a line to continue inside a list item.

- [ ] **For Ex 255** (`- one\n\n two\n`): The `two` has 1 space of indent, but the list item's padding is 2 (marker `-` + 1 space). So `two` should NOT be inside the list item. Check if the padding is computed correctly.

- [ ] Apply fixes
- [ ] **Run batch:** `go test -v -run "TestSpec/Example_(255|257|260|276)$" .`
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 4.4: Fix lazy continuation in blockquotes (Ex 292, 293, 303)

- [ ] **For Ex 303** (`Foo\n- bar\n- baz`): The `- bar` should interrupt the paragraph `Foo`. Check `parseListMarker` — it already has paragraph interruption logic, but may be too restrictive.

- [ ] **For Ex 292/293**: Lazy continuation across blockquote + list boundaries. Check `addTextToContainer` — the lazy continuation check at lines 564-568 should allow continuation into a paragraph inside a blockquote inside a list item.

- [ ] Apply fixes
- [ ] **Run batch:** `go test -v -run "TestSpec/Example_(292|293|303)$" .`
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 4.5: Commit

- [ ] `git add internal/parser/blocks.go && git commit -m "Fix list item creation, continuation padding, and lazy continuation"`

---

## Task 5: Fix Tight/Loose List Detection (depends on Task 4)

After Task 4 fixes create the correct tree structure, tight/loose detection determines whether items get wrapped in `<p>` tags.

**Files:**
- Modify: `internal/parser/blocks.go` — `finalize` list section (lines 740-768)
- Test: `go test -v -run "TestSpec/Example_(308|315|317|318|320|321|325|326)$" .`

### Step 5.1: Replace tight/loose logic with C reference algorithm

- [ ] **Replace** the current tight/loose detection (lines 740-768) with the exact C algorithm:

```go
if node.Type == ast.NodeList {
    node.ListData.Tight = true
    for item := node.FirstChild; item != nil; item = item.Next {
        // check for non-final non-empty list item ending with blank line
        if lastLineBlank(item) && item.Next != nil {
            node.ListData.Tight = false
            break
        }
        // recurse into children of list item
        for subitem := item.FirstChild; subitem != nil; subitem = subitem.Next {
            if (item.Next != nil || subitem.Next != nil) && endsWithBlankLine(subitem) {
                node.ListData.Tight = false
                break
            }
        }
        if !node.ListData.Tight {
            break
        }
    }
}
```

Remove the extra conditions for HTMLBlock, CodeBlock, Heading, and `Paragraph.Next` that are NOT in the C reference.

- [ ] **Run:** `go test -v -run "TestSpec/Example_(308|315|317|318|325|326)$" .`
- [ ] **Run full suite** and confirm ≥ 615 passing

### Step 5.2: Fix lastLineBlank propagation

- [ ] **Check `addTextToContainer`** — the `lastLineBlank` calculation at lines 582-587 must match the C reference exactly. Key conditions:
  - Block quotes are never blank
  - Fenced code blocks are never blank (for tight/loose purposes)
  - Empty list items on their opening line are never blank

- [ ] Verify `setLastLineBlank` is called on the correct nodes. The C code sets it on `container->last_child` when blank, and clears it on all ancestors.

- [ ] **Run full suite** and confirm improvement
- [ ] **Commit:** `git commit -m "Fix tight/loose list detection to match C reference algorithm"`

---

## Verification

After all tasks are complete:

- [ ] `go clean -testcache && go test -v -run TestSpec . 2>&1 | grep -c "PASS: TestSpec/"` — expect 652
- [ ] `go test ./...` — all tests pass
- [ ] `go build ./...` — clean build
- [ ] Update `CLAUDE.md` spec parity line to `652/652 (100%)`
