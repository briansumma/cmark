package parser

import (
	"strings"

	"github.com/briansumma/cmark/internal/scanners"
	"github.com/briansumma/cmark/pkg/ast"
)

type subject struct {
	input        []byte
	pos          int
	line         int
	blockOffset  int
	columnOffset   int
	refmap       *ast.ReferenceMap
}

func newSubject(input []byte, line, blockOffset, columnOffset int, refmap *ast.ReferenceMap) *subject {
	return &subject{
		input:       input,
		pos:         0,
		line:        line,
		blockOffset: blockOffset,
		columnOffset:  columnOffset,
		refmap:      refmap,
	}
}

func (s *subject) peekChar() byte {
	if s.pos >= len(s.input) {
		return 0
	}
	return s.input[s.pos]
}

func (s *subject) peekAt(pos int) byte {
	if pos >= len(s.input) {
		return 0
	}
	return s.input[pos]
}

func (s *subject) isEOF() bool {
	return s.pos >= len(s.input)
}

func (s *subject) skipSpaces() bool {
	skipped := false
	for !s.isEOF() && (s.peekChar() == ' ' || s.peekChar() == '\t') {
		s.pos++
		skipped = true
	}
	return skipped
}

func (s *subject) skipLineEnd() bool {
	if s.peekChar() == '\r' {
		s.pos++
	}
	if s.peekChar() == '\n' {
		s.pos++
		return true
	}
	return false
}

func (s *subject) advance(n int) {
	s.pos += n
}

func (s *subject) makeNode(t ast.NodeType, startColumn, endColumn int) *ast.Node {
	n := ast.NewNode(t)
	n.StartLine = s.line
	n.EndLine = s.line
	n.StartColumn = startColumn + 1 + s.columnOffset + s.blockOffset
	n.EndColumn = endColumn + 1 + s.columnOffset + s.blockOffset
	return n
}

func parseInlines(parent *ast.Node, refmap *ast.ReferenceMap, options ast.Options) {
	if parent.Data == "" {
		return
	}
	// For now, just split on newlines and create text/softbreak nodes.
	// A full inline parser would handle emphasis, links, code spans, etc.
	lines := strings.Split(parent.Data, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i, line := range lines {
		if i > 0 {
			parent.AppendChild(ast.NewNode(ast.NodeSoftbreak))
		}
		if line != "" {
			textNode := ast.NewNode(ast.NodeText)
			textNode.Data = line
			parent.AppendChild(textNode)
		}
	}
	parent.Data = ""
}

// parseInlinesFull is a more complete inline parser.
// For now it just calls the simple version above.
func parseInlinesFull(parent *ast.Node, refmap *ast.ReferenceMap, options ast.Options) {
	if parent.Data == "" {
		return
	}
	subj := newSubject([]byte(parent.Data), parent.StartLine, parent.StartColumn-1, 0, refmap)
	for !subj.isEOF() {
		n := parseInline(subj, parent, options)
		if n == nil {
			break
		}
	}
	parent.Data = ""
}

func parseInline(subj *subject, parent *ast.Node, options ast.Options) *ast.Node {
	c := subj.peekChar()

	switch c {
	case '\\':
		return handleBackslash(subj, parent)
	case '\r', '\n':
		return handleNewline(subj, parent)
	case '`':
		return handleBackticks(subj, parent)
	case '*', '_':
		return handleDelim(subj, parent, c, options&ast.OptSmart != 0)
	case '[', ']':
		return handleBracket(subj, parent, c)
	case '<':
		return handlePointyBrace(subj, parent, options)
	case '&':
		return handleEntity(subj, parent)
	default:
		return handleText(subj, parent)
	}
}

func handleBackslash(subj *subject, parent *ast.Node) *ast.Node {
	subj.pos++ // skip backslash
	if subj.isEOF() {
		textNode := ast.NewNode(ast.NodeText)
		textNode.Data = "\\"
		parent.AppendChild(textNode)
		return textNode
	}
	c := subj.peekChar()
	if c == '\r' || c == '\n' {
		// hard line break
		subj.skipLineEnd()
		br := ast.NewNode(ast.NodeLinebreak)
		parent.AppendChild(br)
		return br
	}
	if isPunct(c) {
		subj.pos++
		textNode := ast.NewNode(ast.NodeText)
		textNode.Data = string(c)
		parent.AppendChild(textNode)
		return textNode
	}
	// not a valid escape, keep backslash as text
	textNode := ast.NewNode(ast.NodeText)
	textNode.Data = "\\"
	parent.AppendChild(textNode)
	return textNode
}

func handleNewline(subj *subject, parent *ast.Node) *ast.Node {
	subj.skipLineEnd()
	br := ast.NewNode(ast.NodeSoftbreak)
	parent.AppendChild(br)
	return br
}

func handleBackticks(subj *subject, parent *ast.Node) *ast.Node {
	startPos := subj.pos
	backticks := 0
	for subj.peekChar() == '`' {
		subj.pos++
		backticks++
	}

	// Find closing backticks
	closePos := subj.pos
	found := false
	for closePos < len(subj.input) {
		if subj.input[closePos] == '`' {
			count := 0
			for closePos+count < len(subj.input) && subj.input[closePos+count] == '`' {
				count++
			}
			if count >= backticks {
				found = true
				closePos += count
				break
			}
			closePos += count
		} else {
			closePos++
		}
	}

	if !found {
		// No closing backticks, treat as text
		subj.pos = startPos + backticks
		textNode := ast.NewNode(ast.NodeText)
		textNode.Data = strings.Repeat("`", backticks)
		parent.AppendChild(textNode)
		return textNode
	}

	code := string(subj.input[startPos+backticks : closePos-backticks])
	// Trim leading/trailing spaces if both sides have spaces
	code = strings.TrimSpace(code)
	codeNode := ast.NewNode(ast.NodeCode)
	codeNode.Data = code
	parent.AppendChild(codeNode)
	subj.pos = closePos
	return codeNode
}

func handleDelim(subj *subject, parent *ast.Node, c byte, smart bool) *ast.Node {
	count := 0
	for subj.peekChar() == c {
		subj.pos++
		count++
	}

	textNode := ast.NewNode(ast.NodeText)
	textNode.Data = strings.Repeat(string(c), count)
	parent.AppendChild(textNode)
	return textNode
}

func handleBracket(subj *subject, parent *ast.Node, c byte) *ast.Node {
	subj.pos++
	textNode := ast.NewNode(ast.NodeText)
	textNode.Data = string(c)
	parent.AppendChild(textNode)
	return textNode
}

func handlePointyBrace(subj *subject, parent *ast.Node, options ast.Options) *ast.Node {
	subj.pos++

	// Try autolink
	autolinkLen := scanners.ScanAutolinkURI(subj.input[subj.pos:])
	if autolinkLen > 0 {
		url := string(subj.input[subj.pos : subj.pos+autolinkLen-1])
		subj.pos += autolinkLen
		link := ast.NewNode(ast.NodeLink)
		link.LinkData = &ast.LinkData{URL: url}
		parent.AppendChild(link)
		return link
	}

	// Try autolink email
	autolinkLen = scanners.ScanAutolinkEmail(subj.input[subj.pos:])
	if autolinkLen > 0 {
		email := string(subj.input[subj.pos : subj.pos+autolinkLen-1])
		subj.pos += autolinkLen
		link := ast.NewNode(ast.NodeLink)
		link.LinkData = &ast.LinkData{URL: "mailto:" + email}
		parent.AppendChild(link)
		return link
	}

	// Try HTML tag
	tagLen := scanners.ScanHTMLTag(subj.input[subj.pos:])
	if tagLen > 0 {
		tag := string(subj.input[subj.pos : subj.pos+tagLen])
		subj.pos += tagLen
		htmlNode := ast.NewNode(ast.NodeHTMLInline)
		htmlNode.Data = "<" + tag
		parent.AppendChild(htmlNode)
		return htmlNode
	}

	// Not a valid construct, treat as text
	textNode := ast.NewNode(ast.NodeText)
	textNode.Data = "<"
	parent.AppendChild(textNode)
	return textNode
}

func handleEntity(subj *subject, parent *ast.Node) *ast.Node {
	startPos := subj.pos
	subj.pos++ // skip &

	// Try numeric entity
	if subj.peekChar() == '#' {
		subj.pos++
		isHex := false
		if subj.peekChar() == 'x' || subj.peekChar() == 'X' {
			isHex = true
			subj.pos++
		}
		numStart := subj.pos
		for isHex {
			c := subj.peekChar()
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
				subj.pos++
			} else {
				break
			}
		}
		for !isHex {
			c := subj.peekChar()
			if c >= '0' && c <= '9' {
				subj.pos++
			} else {
				break
			}
		}
		if subj.pos > numStart && subj.peekChar() == ';' {
			subj.pos++
			htmlNode := ast.NewNode(ast.NodeHTMLInline)
			htmlNode.Data = string(subj.input[startPos:subj.pos])
			parent.AppendChild(htmlNode)
			return htmlNode
		}
	}

	// Try named entity
	nameStart := subj.pos
	for {
		c := subj.peekChar()
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			subj.pos++
		} else {
			break
		}
	}
	if subj.pos > nameStart && subj.peekChar() == ';' {
		subj.pos++
		htmlNode := ast.NewNode(ast.NodeHTMLInline)
		htmlNode.Data = string(subj.input[startPos:subj.pos])
		parent.AppendChild(htmlNode)
		return htmlNode
	}

	// Not a valid entity
	subj.pos = startPos
	textNode := ast.NewNode(ast.NodeText)
	textNode.Data = "&"
	parent.AppendChild(textNode)
	return textNode
}

func handleText(subj *subject, parent *ast.Node) *ast.Node {
	startPos := subj.pos
	for !subj.isEOF() {
		c := subj.peekChar()
		if c == '\\' || c == '\r' || c == '\n' || c == '`' ||
			c == '*' || c == '_' || c == '[' || c == ']' ||
			c == '<' || c == '&' {
			break
		}
		subj.pos++
	}
	textNode := ast.NewNode(ast.NodeText)
	textNode.Data = string(subj.input[startPos:subj.pos])
	parent.AppendChild(textNode)
	return textNode
}

func isPunct(c byte) bool {
	return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
		(c >= '[' && c <= '`') || (c >= '{' && c <= '~')
}
