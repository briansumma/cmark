// Package parser implements the CommonMark block and inline parser.
package parser

import (
	"io"

	"github.com/briansumma/cmark/internal/buffer"
	"github.com/briansumma/cmark/pkg/ast"
)

// Parser holds the state of a streaming parser.
type Parser struct {
	root      *ast.Node
	current   *ast.Node
	refmap    *ast.ReferenceMap
	options   ast.Options
	curline   buffer.Buffer
	linebuf   buffer.Buffer
	content   buffer.Buffer
	lineNum   int
	offset    int
	column    int
	indent    int
	blank     bool
	totalSize uint
}

// New creates a new parser with the given options.
func New(options ast.Options) *Parser {
	return &Parser{
		root:    ast.NewNode(ast.NodeDocument),
		refmap:  ast.NewReferenceMap(),
		options: options,
	}
}

// Feed feeds a chunk of input to the parser.
func (p *Parser) Feed(data []byte) {
	// TODO: stub
}

// Finish finishes parsing and returns the document tree.
func (p *Parser) Finish() *ast.Node {
	// TODO: stub
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
	// TODO: stub
	return ast.NewNode(ast.NodeDocument), nil
}

// ParseInlines parses inline content inside a leaf block.
func ParseInlines(parent *ast.Node, refmap *ast.ReferenceMap, options ast.Options) {
	// TODO: stub
}

// ParseReferenceInline parses reference definitions from a chunk.
func ParseReferenceInline(input []byte, refmap *ast.ReferenceMap) int {
	// TODO: stub
	return 0
}
