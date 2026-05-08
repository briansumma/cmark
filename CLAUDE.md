# cmark — Go rewrite of CommonMark reference parser

Full Go rewrite of the C99 `cmark` CommonMark implementation (spec v0.31.2).

## Build & Test

```bash
go build ./...                          # build everything
go test ./...                           # unit tests
go test -run TestSpec .                 # all 652 spec tests
go test -run "TestSpec/Example_123$" . # single spec example
go run ./cmd/cmark                      # CLI (reads stdin)
```

## Architecture

```
pkg/ast/                 node types, tree ops, iterator, reference map
internal/
  parser/                block + inline parser (blocks.go, inlines.go, parser.go)
  render/
    html/                HTML output
    xml/                 XML AST output
    commonmark/          CommonMark roundtrip
    man/                 groff man
    latex/               LaTeX
  buffer/                growable string buffer
  utf8/                  UTF-8 validation, case folding
  ctype/                 locale-independent char classification
  scanners/              re2c-generated pattern matchers (Go port)
cmd/cmark/               CLI executable
spec.json                CommonMark 0.31.2 spec tests (652 examples)
```

## Spec Parity

- **Branch:** `feature/golang`
- **Plan:** `~/.claude/plans/fuzzy-bubbling-feather.md`
- **Current:** ~607/652 (93.1%)
- **Remaining gaps:** list tight/loose detection (26 tests), link ref-def edge cases

## Gotchas

- **List tight/loose detection** in `blocks.go:finalize()` — highly sensitive to blank-line and
  sublist interactions. Changes here routinely cause regressions.
- **HTML block types** (1–7) have different termination rules; the scanners in
  `internal/scanners/` handle blocking tagging and inline detection separately.
- **Backslash escapes in link labels** are resolved during `normalizeReference` in
  `pkg/ast/ast.go`, not during inline parsing.
- **`noLinkOpeners`** prevents nested unmatched brackets from resolving — set after
  unmatched `]` in `handleCloseBracket`.
- Trailing spaces in paragraph content are stripped in `finalize()`, matching the
  spec rule that trailing spaces are ignored.
