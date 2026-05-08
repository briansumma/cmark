package parser

import (
	"fmt"
	"html"
	"strings"

	"github.com/briansumma/cmark/internal/scanners"
	cmarkutf8 "github.com/briansumma/cmark/internal/utf8"
	"github.com/briansumma/cmark/pkg/ast"
)

type subject struct {
	input         []byte
	pos           int
	line          int
	blockOffset   int
	columnOffset  int
	refmap        *ast.ReferenceMap
	lastDelim     *delimiter
	lastBracket   *bracket
	noLinkOpeners bool
}

func newSubject(input []byte, line, blockOffset, columnOffset int, refmap *ast.ReferenceMap) *subject {
	return &subject{
		input:         input,
		pos:           0,
		line:          line,
		blockOffset:   blockOffset,
		columnOffset:  columnOffset,
		refmap:        refmap,
		noLinkOpeners: true,
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

func makeStr(subj *subject, startColumn, endColumn int, text string) *ast.Node {
	n := subj.makeNode(ast.NodeText, startColumn, endColumn)
	n.Data = text
	return n
}

func parseInlines(parent *ast.Node, refmap *ast.ReferenceMap, options ast.Options) {
	parseInlinesFull(parent, refmap, options)
}

func parseInlinesFull(parent *ast.Node, refmap *ast.ReferenceMap, options ast.Options) {
	if parent.Data == "" {
		return
	}
	subj := newSubject([]byte(parent.Data), parent.StartLine, parent.StartColumn-1, 0, refmap)
	for !subj.isEOF() {
		if !parseInline(subj, parent, options) {
			break
		}
	}
	processEmphasis(subj, 0)
	for subj.lastDelim != nil {
		subj.removeDelimiter(subj.lastDelim)
	}
	for subj.lastBracket != nil {
		subj.popBracket()
	}
	parent.Data = ""
}

func parseInline(subj *subject, parent *ast.Node, options ast.Options) bool {
	c := subj.peekChar()
	if c == 0 {
		return false
	}
	var newInl *ast.Node
	switch c {
	case '\\':
		newInl = handleBackslash(subj, parent)
	case '\r', '\n':
		newInl = handleNewline(subj, parent)
	case '`':
		newInl = handleBackticks(subj, parent)
	case '*', '_':
		newInl = handleDelim(subj, parent, c, options&ast.OptSmart != 0)
	case '[':
		subj.pos++
		newInl = ast.NewNode(ast.NodeText)
		newInl.Data = "["
		parent.AppendChild(newInl)
		subj.pushBracket(false, newInl)
		return true
	case ']':
		newInl = handleCloseBracket(subj, parent)
	case '<':
		newInl = handlePointyBrace(subj, parent, options)
	case '&':
		newInl = handleEntity(subj, parent)
	case '!':
		subj.pos++
		if subj.peekChar() == '[' {
			subj.pos++
			newInl = ast.NewNode(ast.NodeText)
			newInl.Data = "!["
			parent.AppendChild(newInl)
			subj.pushBracket(true, newInl)
			return true
		}
		newInl = ast.NewNode(ast.NodeText)
		newInl.Data = "!"
		parent.AppendChild(newInl)
		return true
	default:
		newInl = handleText(subj, parent)
	}
	_ = newInl
	return true
}

func handleBackslash(subj *subject, parent *ast.Node) *ast.Node {
	subj.pos++ // skip backslash
	if subj.isEOF() {
		n := ast.NewNode(ast.NodeText)
		n.Data = "\\"
		parent.AppendChild(n)
		return n
	}
	c := subj.peekChar()
	if c == '\r' || c == '\n' {
		subj.skipLineEnd()
		subj.skipSpaces()
		br := ast.NewNode(ast.NodeLinebreak)
		parent.AppendChild(br)
		return br
	}
	if isPunct(c) {
		subj.pos++
		n := ast.NewNode(ast.NodeText)
		n.Data = string(c)
		parent.AppendChild(n)
		return n
	}
	n := ast.NewNode(ast.NodeText)
	n.Data = "\\"
	parent.AppendChild(n)
	return n
}

func handleNewline(subj *subject, parent *ast.Node) *ast.Node {
	nlpos := subj.pos
	if subj.peekChar() == '\r' {
		subj.pos++
	}
	if subj.peekChar() == '\n' {
		subj.pos++
	}
	subj.line++
	subj.columnOffset = -subj.pos
	subj.skipSpaces()
	if nlpos >= 2 && subj.peekAt(nlpos-1) == ' ' && subj.peekAt(nlpos-2) == ' ' {
		br := ast.NewNode(ast.NodeLinebreak)
		parent.AppendChild(br)
		return br
	}
	br := ast.NewNode(ast.NodeSoftbreak)
	parent.AppendChild(br)
	return br
}

func normalizeCode(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	containsNonSpace := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\r' {
			if i+1 < len(s) && s[i+1] == '\n' {
				continue
			}
			c = ' '
		} else if c == '\n' {
			c = ' '
		}
		b.WriteByte(c)
		if c != ' ' {
			containsNonSpace = true
		}
	}
	result := b.String()
	if containsNonSpace && len(result) > 0 && result[0] == ' ' && result[len(result)-1] == ' ' {
		return result[1 : len(result)-1]
	}
	return result
}

func handleBackticks(subj *subject, parent *ast.Node) *ast.Node {
	startPos := subj.pos
	backticks := 0
	for subj.peekChar() == '`' {
		subj.pos++
		backticks++
	}

	closePos := subj.pos
	found := false
	for closePos < len(subj.input) {
		if subj.input[closePos] == '`' {
			count := 0
			for closePos+count < len(subj.input) && subj.input[closePos+count] == '`' {
				count++
			}
			if count == backticks {
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
		subj.pos = startPos + backticks
		n := ast.NewNode(ast.NodeText)
		n.Data = strings.Repeat("`", backticks)
		parent.AppendChild(n)
		return n
	}

	code := string(subj.input[startPos+backticks : closePos-backticks])
	code = normalizeCode(code)
	n := ast.NewNode(ast.NodeCode)
	n.Data = code
	parent.AppendChild(n)
	subj.pos = closePos
	return n
}

func (s *subject) scanDelims(c byte, canOpen, canClose *bool) int {
	numdelims := 0
	var beforeChar rune = '\n'
	var afterChar rune = '\n'

	if s.pos > 0 {
		beforePos := s.pos - 1
		for beforePos > 0 && (s.input[beforePos]&0xC0) == 0x80 {
			beforePos--
		}
		r, _ := cmarkutf8.Iterate(s.input[beforePos:])
		beforeChar = r
	}

	if c == '\'' || c == '"' {
		numdelims = 1
		s.pos++
	} else {
		for s.peekChar() == c {
			numdelims++
			s.pos++
		}
	}

	if s.pos < len(s.input) {
		r, _ := cmarkutf8.Iterate(s.input[s.pos:])
		afterChar = r
	}

	leftFlanking := numdelims > 0 && !cmarkutf8.IsSpace(afterChar) &&
		(!cmarkutf8.IsPunctuationOrSymbol(afterChar) ||
			cmarkutf8.IsSpace(beforeChar) || cmarkutf8.IsPunctuationOrSymbol(beforeChar))
	rightFlanking := numdelims > 0 && !cmarkutf8.IsSpace(beforeChar) &&
		(!cmarkutf8.IsPunctuationOrSymbol(beforeChar) ||
			cmarkutf8.IsSpace(afterChar) || cmarkutf8.IsPunctuationOrSymbol(afterChar))

	if c == '_' {
		*canOpen = leftFlanking && (!rightFlanking || cmarkutf8.IsPunctuationOrSymbol(beforeChar))
		*canClose = rightFlanking && (!leftFlanking || cmarkutf8.IsPunctuationOrSymbol(afterChar))
	} else if c == '*' {
		*canOpen = leftFlanking
		*canClose = rightFlanking
	} else {
		*canOpen = false
		*canClose = false
	}

	return numdelims
}

func handleDelim(subj *subject, parent *ast.Node, c byte, smart bool) *ast.Node {
	var canOpen, canClose bool
	numdelims := subj.scanDelims(c, &canOpen, &canClose)

	contents := strings.Repeat(string(c), numdelims)
	n := ast.NewNode(ast.NodeText)
	n.Data = contents
	parent.AppendChild(n)

	if canOpen || canClose {
		subj.pushDelimiter(c, canOpen, canClose, n)
	}
	return n
}

func linkLabel(subj *subject) (string, bool) {
	startPos := subj.pos
	if subj.peekChar() != '[' {
		return "", false
	}
	subj.pos++
	length := 0
	for {
		c := subj.peekChar()
		if c == 0 || c == '[' || c == ']' {
			break
		}
		if c == '\\' {
			subj.pos++
			length++
			if isPunct(subj.peekChar()) {
				subj.pos++
				length++
			}
		} else {
			subj.pos++
			length++
		}
		if length > ast.MaxLinkLabelLength {
			subj.pos = startPos
			return "", false
		}
	}
	if subj.peekChar() == ']' {
		label := string(subj.input[startPos+1 : subj.pos])
		subj.pos++ // advance past ]
		return label, true
	}
	subj.pos = startPos
	return "", false
}

func scanLinkURL(input []byte, offset int) (int, []byte) {
	if offset >= len(input) {
		return -1, nil
	}
	if input[offset] == '<' {
		return scanLinkURLAngle(input, offset)
	}
	return scanLinkURLPlain(input, offset)
}

func scanLinkURLAngle(input []byte, offset int) (int, []byte) {
	i := offset + 1
	for i < len(input) {
		if input[i] == '>' {
			return i + 1 - offset, input[offset+1 : i]
		}
		if input[i] == '\\' && i+1 < len(input) {
			i += 2
			continue
		}
		if input[i] == '\n' || input[i] == '<' {
			return -1, nil
		}
		i++
	}
	return -1, nil
}

func scanLinkURLPlain(input []byte, offset int) (int, []byte) {
	i := offset
	nbP := 0
	for i < len(input) {
		if input[i] == '\\' && i+1 < len(input) && isPunct(input[i+1]) {
			i += 2
			continue
		}
		if input[i] == '(' {
			nbP++
			i++
			if nbP > 32 {
				return -1, nil
			}
			continue
		}
		if input[i] == ')' {
			if nbP == 0 {
				break
			}
			nbP--
			i++
			continue
		}
		if input[i] == ' ' || input[i] == '\t' || input[i] == '\r' || input[i] == '\n' {
			if i == offset {
				return -1, nil
			}
			break
		}
		i++
	}
	if i >= len(input) || nbP != 0 {
		return -1, nil
	}
	return i - offset, input[offset:i]
}

func cleanURL(urlStr string) string {
	urlStr = strings.TrimSpace(urlStr)
	// unescape backslashes
	var b strings.Builder
	b.Grow(len(urlStr))
	for i := 0; i < len(urlStr); i++ {
		if urlStr[i] == '\\' && i+1 < len(urlStr) && isPunct(urlStr[i+1]) {
			b.WriteByte(urlStr[i+1])
			i++
		} else {
			b.WriteByte(urlStr[i])
		}
	}
	// decode entities, then percent-encode non-ASCII
	raw := b.String()
	decoded := html.UnescapeString(raw)
	return percentEncodeURL(decoded)
}


func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func percentEncodeURL(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			b.WriteByte('%')
			b.WriteByte(s[i+1])
			b.WriteByte(s[i+2])
			i += 2
		} else if isURLSafe(s[i]) && s[i] != '%' {
			b.WriteByte(s[i])
		} else {
			b.WriteString(fmt.Sprintf("%%%02X", s[i]))
		}
	}
	return b.String()
}

func isURLSafe(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' ||
		c == '~' || c == '/' || c == ':' || c == '@' || c == '!' ||
		c == '$' || c == '&' || c == '\'' || c == '(' || c == ')' ||
		c == '*' || c == '+' || c == ',' || c == ';' || c == '=' ||
		c == '?' || c == '#'
}

func cleanTitle(title string) string {
	if len(title) == 0 {
		return ""
	}
	first := title[0]
	last := title[len(title)-1]
	if (first == '\'' && last == '\'') || (first == '"' && last == '"') ||
		(first == '(' && last == ')') {
		title = title[1 : len(title)-1]
	}
	var b strings.Builder
	b.Grow(len(title))
	for i := 0; i < len(title); i++ {
		if title[i] == '\\' && i+1 < len(title) && isPunct(title[i+1]) {
			b.WriteByte(title[i+1])
			i++
		} else {
			b.WriteByte(title[i])
		}
	}
	// decode HTML entities
	return html.UnescapeString(b.String())
}

func handleCloseBracket(subj *subject, parent *ast.Node) *ast.Node {
	subj.pos++ // advance past ]
	initialPos := subj.pos

	opener := subj.lastBracket
	if opener == nil {
		n := makeStr(subj, subj.pos-1, subj.pos-1, "]")
		parent.AppendChild(n)
		return n
	}

	isImage := opener.image
	if !opener.active {
		subj.popBracket()
		n := makeStr(subj, subj.pos-1, subj.pos-1, "]")
		parent.AppendChild(n)
		return n
	}

	afterLinkTextPos := subj.pos
	matched := false
	var urlStr, titleStr string

	// Try inline link
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
			posBeforeLabel := subj.pos
		rawLabel, foundLabel = linkLabel(subj)
		if !foundLabel {
			subj.pos = posBeforeLabel
		}
		if (!foundLabel || rawLabel == "") && !opener.bracketAfter {
			rawLabel = string(subj.input[opener.position:initialPos-1])
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

	if !matched {
		subj.popBracket()
		subj.pos = initialPos
		n := makeStr(subj, subj.pos-1, subj.pos-1, "]")
		parent.AppendChild(n)
		return n
	}

	var inl *ast.Node
	if isImage {
		inl = ast.NewNode(ast.NodeImage)
	} else {
		inl = ast.NewNode(ast.NodeLink)
	}
	inl.LinkData = &ast.LinkData{URL: urlStr, Title: titleStr}

	// Move nodes between opener and closer into link
	tmp := opener.node.Next
	for tmp != nil {
		tmpnext := tmp.Next
		tmp.Unlink()
		inl.AppendChild(tmp)
		tmp = tmpnext
	}

	opener.node.InsertBefore(inl)
	opener.node.Unlink()

	processEmphasis(subj, opener.position)
	subj.popBracket()

	// If this was a link (not image), deactivate earlier link brackets
	// to prevent nested links.
	if !isImage {
		for b := subj.lastBracket; b != nil; b = b.prev {
			if !b.image {
				b.active = false
			}
		}
	}
	return nil
}

func insertEmph(subj *subject, opener, closer *delimiter) *delimiter {
	openerInl := opener.inl
	closerInl := closer.inl
	openerNumChars := len(openerInl.Data)
	closerNumChars := len(closerInl.Data)

	useDelims := 1
	if closerNumChars >= 2 && openerNumChars >= 2 {
		useDelims = 2
	}

	// Trim delimiter text
	openerInl.Data = openerInl.Data[:openerNumChars-useDelims]
	closerInl.Data = closerInl.Data[useDelims:]

	// Remove delimiters between opener and closer
	delim := closer.prev
	for delim != nil && delim != opener {
		tmpDelim := delim.prev
		subj.removeDelimiter(delim)
		delim = tmpDelim
	}

	// Create emph or strong node
	var emph *ast.Node
	if useDelims == 1 {
		emph = ast.NewNode(ast.NodeEmph)
	} else {
		emph = ast.NewNode(ast.NodeStrong)
	}

	// Splice nodes between opener and closer into emph
	tmp := openerInl.Next
	if tmp != nil && tmp != closerInl {
		emph.FirstChild = tmp
		tmp.Prev = nil
		for tmp != nil && tmp != closerInl {
			tmpnext := tmp.Next
			tmp.Parent = emph
			if tmpnext == closerInl {
				emph.LastChild = tmp
				tmp.Next = nil
			}
			tmp = tmpnext
		}
	}

	openerInl.Next = emph
	closerInl.Prev = emph
	emph.Prev = openerInl
	emph.Next = closerInl
	emph.Parent = openerInl.Parent

	if len(openerInl.Data) == 0 {
		openerInl.Unlink()
		subj.removeDelimiter(opener)
	}
	var nextDelim *delimiter
	if len(closerInl.Data) == 0 {
		closerInl.Unlink()
		nextDelim = closer.next
		subj.removeDelimiter(closer)
		closer = nextDelim
	}
	return closer
}

func processEmphasis(subj *subject, stackBottom int) {
	var openersBottom [15]int
	for i := range openersBottom {
		openersBottom[i] = stackBottom
	}

	var closer *delimiter
	candidate := subj.lastDelim
	for candidate != nil && candidate.position >= stackBottom {
		closer = candidate
		candidate = candidate.prev
	}

	for closer != nil {
		if closer.canClose {
			openersBottomIndex := 0
			switch closer.char {
			case '"':
				openersBottomIndex = 0
			case '\'':
				openersBottomIndex = 1
			case '_':
				openersBottomIndex = 2
				if closer.canOpen {
					openersBottomIndex += 3
				}
				openersBottomIndex += closer.origNum % 3
			case '*':
				openersBottomIndex = 8
				if closer.canOpen {
					openersBottomIndex += 3
				}
				openersBottomIndex += closer.origNum % 3
			}

			opener := closer.prev
			openerFound := false
			for opener != nil && opener.position >= openersBottom[openersBottomIndex] {
				if opener.canOpen && opener.char == closer.char {
					if !(closer.canOpen || opener.canClose) ||
						closer.origNum%3 == 0 ||
						(opener.origNum+closer.origNum)%3 != 0 {
						openerFound = true
						break
					}
				}
				opener = opener.prev
			}
			oldCloser := closer
			if closer.char == '*' || closer.char == '_' {
				if openerFound {
					closer = insertEmph(subj, opener, closer)
				} else {
					closer = closer.next
				}
			}
			if !openerFound {
				openersBottom[openersBottomIndex] = oldCloser.position
				if !oldCloser.canOpen {
					subj.removeDelimiter(oldCloser)
				}
			}
		} else {
			closer = closer.next
		}
	}

	for subj.lastDelim != nil && subj.lastDelim.position >= stackBottom {
		subj.removeDelimiter(subj.lastDelim)
	}
}

func handlePointyBrace(subj *subject, parent *ast.Node, options ast.Options) *ast.Node {
	startPos := subj.pos
	subj.pos++

	autolinkLen := scanners.ScanAutolinkURI(subj.input[subj.pos:])
	if autolinkLen > 0 {
		url := string(subj.input[subj.pos : subj.pos+autolinkLen-1])
		subj.pos += autolinkLen
		link := ast.NewNode(ast.NodeLink)
		link.LinkData = &ast.LinkData{URL: percentEncodeURL(url)}
		text := ast.NewNode(ast.NodeText)
		text.Data = url
		link.AppendChild(text)
		parent.AppendChild(link)
		return link
	}

	autolinkLen = scanners.ScanAutolinkEmail(subj.input[subj.pos:])
	if autolinkLen > 0 {
		email := string(subj.input[subj.pos : subj.pos+autolinkLen-1])
		subj.pos += autolinkLen
		link := ast.NewNode(ast.NodeLink)
		link.LinkData = &ast.LinkData{URL: "mailto:" + email}
		text := ast.NewNode(ast.NodeText)
		text.Data = email
		link.AppendChild(text)
		parent.AppendChild(link)
		return link
	}

	tagLen := scanners.ScanHTMLTag(subj.input[subj.pos:])
	if tagLen > 0 {
		subj.pos += tagLen
		htmlNode := ast.NewNode(ast.NodeHTMLInline)
		htmlNode.Data = string(subj.input[startPos:subj.pos])
		parent.AppendChild(htmlNode)
		return htmlNode
	}

	// HTML comment: <!-- ... -->
	if subj.peekChar() == '!' && subj.peekAt(subj.pos+1) == '-' &&
		subj.peekAt(subj.pos+2) == '-' {
		m := 0
		// empty comment: <!--> or <!---> ?
		if subj.peekAt(subj.pos+3) == '>' {
			m = 1 + 3 // ! + --->
		} else if subj.peekAt(subj.pos+3) == '-' && subj.peekAt(subj.pos+4) == '>' {
			m = 1 + 4 // ! + --->
		} else {
			n := scanners.ScanHTMLComment(subj.input[subj.pos+1:])
			if n > 0 {
				m = 1 + n // ! + scanned chars (starts at --)
			}
		}
		if m > 0 {
			subj.pos += m
			htmlNode := ast.NewNode(ast.NodeHTMLInline)
			htmlNode.Data = string(subj.input[startPos:subj.pos])
			parent.AppendChild(htmlNode)
			return htmlNode
		}
	}

	// Processing instruction: <? ... ?>
	if subj.peekChar() == '?' {
		m := scanners.ScanHTMLPI(subj.input[subj.pos+1:])
		if m > 0 {
			subj.pos += 1 + m
			htmlNode := ast.NewNode(ast.NodeHTMLInline)
			htmlNode.Data = string(subj.input[startPos:subj.pos])
			parent.AppendChild(htmlNode)
			return htmlNode
		}
	}

	// Declaration <! ... > or CDATA <![CDATA[ ... ]]>
	if subj.peekChar() == '!' {
		if subj.peekAt(subj.pos+1) == '[' &&
			strings.HasPrefix(string(subj.input[subj.pos+2:]), "CDATA[") {
			m := scanners.ScanHTMLCDATA(subj.input[subj.pos+2:])
			if m > 0 {
				subj.pos += 2 + m
				htmlNode := ast.NewNode(ast.NodeHTMLInline)
				htmlNode.Data = string(subj.input[startPos:subj.pos])
				parent.AppendChild(htmlNode)
				return htmlNode
			}
		} else {
			m := scanners.ScanHTMLDeclaration(subj.input[subj.pos+1:])
			if m > 0 {
				subj.pos += 1 + m
				htmlNode := ast.NewNode(ast.NodeHTMLInline)
				htmlNode.Data = string(subj.input[startPos:subj.pos])
				parent.AppendChild(htmlNode)
				return htmlNode
			}
		}
	}

	n := ast.NewNode(ast.NodeText)
	n.Data = "<"
	parent.AppendChild(n)
	return n
}

func handleEntity(subj *subject, parent *ast.Node) *ast.Node {
	startPos := subj.pos
	subj.pos++ // skip &

	// Numeric character reference
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
			raw := string(subj.input[numStart:subj.pos-1])
			val, ok := parseEntityNum(raw, isHex)
			if ok && val > 0 && val <= 0x10FFFF &&
				(val < 0xD800 || val > 0xDFFF) {
				decoded := string([]rune{rune(val)})
				n := ast.NewNode(ast.NodeText)
				n.Data = decoded
				parent.AppendChild(n)
			} else if ok && val == 0 {
				n := ast.NewNode(ast.NodeText)
				n.Data = "�"
				parent.AppendChild(n)
			} else {
				subj.pos = startPos + 1
				txt := ast.NewNode(ast.NodeText)
				txt.Data = "&"
				parent.AppendChild(txt)
				return txt
			}
			return nil
		}
				// fall through: numeric without ; or empty → invalid
	}

	// Named entity reference
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
		raw := string(subj.input[startPos:subj.pos])
		decoded := html.UnescapeString(raw)
		n := ast.NewNode(ast.NodeText)
		n.Data = decoded
		parent.AppendChild(n)
		return n
	}

	// Invalid entity – just output &
	subj.pos = startPos + 1
	n := ast.NewNode(ast.NodeText)
	n.Data = "&"
	parent.AppendChild(n)
	return n
}


func parseEntityNum(s string, isHex bool) (int, bool) {
	n := 0
	base := 10
	if isHex {
		base = 16
	}
	for _, c := range s {
		var d int
		switch {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case isHex && c >= 'a' && c <= 'f':
			d = int(c - 'a' + 10)
		case isHex && c >= 'A' && c <= 'F':
			d = int(c - 'A' + 10)
		default:
			return 0, false
		}
		if n > (0x10FFFF-d)/base {
			return 0, false // overflow
		}
		n = n*base + d
	}
	return n, true
}

func handleText(subj *subject, parent *ast.Node) *ast.Node {
	startPos := subj.pos
	for !subj.isEOF() {
		c := subj.peekChar()
		if c == '\\' || c == '\r' || c == '\n' || c == '`' ||
			c == '*' || c == '_' || c == '[' || c == ']' ||
			c == '<' || c == '&' || c == '!' {
			break
		}
		subj.pos++
	}
	text := string(subj.input[startPos:subj.pos])
	if subj.peekChar() == '\r' || subj.peekChar() == '\n' {
		text = strings.TrimRight(text, " ")
	}
	n := ast.NewNode(ast.NodeText)
	n.Data = text
	parent.AppendChild(n)
	return n
}

func isPunct(c byte) bool {
	return (c >= '!' && c <= '/') || (c >= ':' && c <= '@') ||
		(c >= '[' && c <= '`') || (c >= '{' && c <= '~')
}
