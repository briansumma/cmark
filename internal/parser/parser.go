// Package parser implements the CommonMark block and inline parser.
package parser

import (
	"io"
	"strings"

	"github.com/briansumma/cmark/internal/buffer"
	"github.com/briansumma/cmark/internal/scanners"
	"github.com/briansumma/cmark/pkg/ast"
)

const (
	codeIndent = 4
	tabStop    = 4
	maxRefLabelLen = 999
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

// ParseReferenceInline parses a reference link definition at the start of
// input. Returns the number of bytes consumed, or 0 if no definition found.
func ParseReferenceInline(input []byte, refmap *ast.ReferenceMap) int {
	if len(input) < 4 || input[0] != '[' {
		return 0
	}

	// Parse label: scan past [...] to find "]:"
	labelStart := 1
	labelEnd := labelStart
	var labelBuilder strings.Builder
	for labelEnd < len(input) {
		if input[labelEnd] == '\\' && labelEnd+1 < len(input) && isPunct(input[labelEnd+1]) {
			labelBuilder.WriteByte(input[labelEnd+1])
			labelEnd += 2
			continue
		}
		if input[labelEnd] == ']' {
			break
		}
		if input[labelEnd] == '\n' || input[labelEnd] == '\r' || input[labelEnd] == '[' {
			return 0
		}
		labelBuilder.WriteByte(input[labelEnd])
		labelEnd++
	}

	if labelEnd >= len(input) || input[labelEnd] != ']' {
		return 0
	}

	// Check for ":" after "]"
	colonPos := labelEnd + 1
	if colonPos >= len(input) || input[colonPos] != ':' {
		return 0
	}

	label := labelBuilder.String()
	if len(label) == 0 || len(label) > maxRefLabelLen {
		return 0
	}

	// Parse URL after ":"
	URLStart := colonPos + 1
	URLStart += skipSpaces(input[URLStart:])
	if URLStart >= len(input) {
		return 0
	}

	var URL string
	URLEnd := URLStart
	urllen, urlBytes := scanLinkURL(input, URLStart)
	if urlBytes == nil || urllen <= 0 {
		return 0
	}
	URL = cleanURL(string(urlBytes))
	URLEnd = URLStart + urllen

	// Parse optional title
	titleStart := URLEnd
	titleStart += skipSpaces(input[titleStart:])
	title := ""
	if titleStart < len(input) {
		c := input[titleStart]
		if c == '"' || c == '\'' || c == '(' {
			titleLen := scanners.ScanLinkTitle(input[titleStart:])
			if titleLen > 0 {
				title = cleanTitle(string(input[titleStart : titleStart+titleLen]))
				titleStart += titleLen
			}
		}
	}

	// Skip trailing whitespace and blank lines
	end := URLEnd
	if titleStart > URLEnd {
		end = titleStart
	}
	end += skipSpaces(input[end:])
	end += skipBlankLines(input[end:])

	// Add to reference map
	refmap.Create(label, URL, title)

	return end
}

func skipSpaces(input []byte) int {
	i := 0
	for i < len(input) && (input[i] == ' ' || input[i] == '\t') {
		i++
	}
	return i
}

func skipBlankLines(input []byte) int {
	i := 0
	for i < len(input) {
		if input[i] == '\n' {
			i++
			continue
		}
		if input[i] == '\r' && i+1 < len(input) && input[i+1] == '\n' {
			i += 2
			continue
		}
		break
	}
	return i
}


