// Package parser implements the CommonMark block and inline parser.
package parser

import (
	"io"

	"github.com/briansumma/cmark/internal/buffer"
	"github.com/briansumma/cmark/pkg/ast"
)

const (
	codeIndent = 4
	tabStop    = 4
)

// Parser holds the state of a streaming parser.
type Parser struct {
	root                   *ast.Node
	current                *ast.Node
	refmap                 *ast.ReferenceMap
	options                ast.Options
	curline                buffer.Buffer
	linebuf                buffer.Buffer
	content                buffer.Buffer
	lineNum                int
	offset                 int
	column                 int
	firstNonspace          int
	firstNonspaceColumn    int
	thematicBreakKillPos   int
	indent                 int
	blank                  bool
	partiallyConsumedTab   bool
	lastLineLength         int
	lastBufferEndedWithCR  bool
	totalSize              uint
}

// New creates a new parser with the given options.
func New(options ast.Options) *Parser {
	root := ast.NewNode(ast.NodeDocument)
	return &Parser{
		root:    root,
		current: root,
		refmap:  ast.NewReferenceMap(),
		options: options,
	}
}

// Feed feeds a chunk of input to the parser.
func (p *Parser) Feed(data []byte) {
	p.feed(data, true)
}

// Finish finishes parsing and returns the document tree.
func (p *Parser) Finish() *ast.Node {
	if p.linebuf.Len() > 0 {
		p.processLine(p.linebuf.Bytes())
		p.linebuf.Clear()
	}
	p.finalizeDocument()
	ast.ConsolidateTextNodes(p.root)
	return p.root
}

// Free frees the parser (no-op in Go).
func (p *Parser) Free() {}

// ParseDocument parses a complete CommonMark document from a byte slice.
func ParseDocument(data []byte, options ast.Options) (*ast.Node, error) {
	p := New(options)
	p.Feed(data)
	return p.Finish(), nil
}

// ParseFile parses a CommonMark document from a file.
func ParseFile(r io.Reader, options ast.Options) (*ast.Node, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	return ParseDocument(data, options)
}

// ParseInlines parses inline content inside a leaf block.
func ParseInlines(parent *ast.Node, refmap *ast.ReferenceMap, options ast.Options) {
	parseInlines(parent, refmap, options)
}

// ParseReferenceInline parses reference definitions from a chunk.
func ParseReferenceInline(input []byte, refmap *ast.ReferenceMap) int {
	// TODO: stub
	return 0
}

