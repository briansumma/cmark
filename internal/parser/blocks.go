package parser

import (
	"bytes"
	"html"
	"strings"

	"github.com/briansumma/cmark/internal/ctype"
	"github.com/briansumma/cmark/internal/scanners"
	"github.com/briansumma/cmark/internal/utf8"
	"github.com/briansumma/cmark/pkg/ast"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func isLineEndChar(c byte) bool {
	return c == '\n' || c == '\r'
}

func isSpaceOrTab(c byte) bool {
	return c == ' ' || c == '\t'
}

func lastLineBlank(node *ast.Node) bool {
	return (node.Flags & ast.NodeLastLineBlank) != 0
}

func setLastLineBlank(node *ast.Node, isBlank bool) {
	if isBlank {
		node.Flags |= ast.NodeLastLineBlank
	} else {
		node.Flags &= ^uint16(ast.NodeLastLineBlank)
	}
}

func setLastLineChecked(node *ast.Node) {
	node.Flags |= ast.NodeLastLineChecked
}

func lastChildIsOpen(container *ast.Node) bool {
	return container.LastChild != nil && (container.LastChild.Flags&ast.NodeOpen) != 0
}

func acceptsLines(node *ast.Node) bool {
	switch node.Type {
	case ast.NodeParagraph, ast.NodeHeading, ast.NodeCodeBlock:
		return true
	}
	return false
}

func containsInlines(node *ast.Node) bool {
	switch node.Type {
	case ast.NodeParagraph, ast.NodeHeading:
		return true
	}
	return false
}

func isBlank(s []byte, offset int) bool {
	for i := offset; i < len(s); i++ {
		if !isLineEndChar(s[i]) {
			return false
		}
	}
	return true
}

func peekAt(input []byte, n int) byte {
	if n >= len(input) {
		return 0
	}
	return input[n]
}

func findFirstNonspace(p *Parser, input []byte) {
	charsToTab := tabStop - (p.column % tabStop)
	if p.firstNonspace <= p.offset {
		p.firstNonspace = p.offset
		p.firstNonspaceColumn = p.column
		for {
			c := peekAt(input, p.firstNonspace)
			if c == ' ' {
				p.firstNonspace++
				p.firstNonspaceColumn++
				charsToTab--
				if charsToTab == 0 {
					charsToTab = tabStop
				}
			} else if c == '\t' {
				p.firstNonspace++
				p.firstNonspaceColumn += charsToTab
				charsToTab = tabStop
			} else {
				break
			}
		}
	}
	p.indent = p.firstNonspaceColumn - p.column
	p.blank = isLineEndChar(peekAt(input, p.firstNonspace))
}

func advanceOffset(p *Parser, input []byte, count int, columns bool) {
	for count > 0 {
		c := peekAt(input, p.offset)
		if c == 0 {
			break
		}
		if c == '\t' {
			charsToTab := tabStop - (p.column % tabStop)
			if columns {
				p.partiallyConsumedTab = charsToTab > count
				charsToAdvance := charsToTab
				if count < charsToAdvance {
					charsToAdvance = count
				}
				p.column += charsToAdvance
				if !p.partiallyConsumedTab {
					p.offset++
				}
				count -= charsToAdvance
			} else {
				p.partiallyConsumedTab = false
				p.column += charsToTab
				p.offset++
				count--
			}
		} else {
			p.partiallyConsumedTab = false
			p.offset++
			p.column++
			count--
		}
	}
}

func addLine(input []byte, p *Parser) {
	if p.blank && p.current.Type == ast.NodeCodeBlock {
		b := p.current
		if b.CodeData != nil && !b.CodeData.Fenced {
			// omit trailing blank lines in indented code blocks
			return
		}
	}
	if p.partiallyConsumedTab {
		p.offset++
		charsToTab := tabStop - (p.column % tabStop)
		for i := 0; i < charsToTab; i++ {
			p.content.AppendByte(' ')
		}
	}
	if p.offset < len(input) {
		p.content.Append(input[p.offset:])
	}
}

func (p *Parser) removeTrailingBlankLines() {
	s := p.content.String()
	lines := strings.Split(s, "\n")
	i := len(lines)
	for i > 0 && strings.TrimSpace(lines[i-1]) == "" {
		i--
	}
	p.content.Set([]byte(strings.Join(lines[:i], "\n")))
}

func endsWithBlankLine(node *ast.Node) bool {
	if lastLineBlank(node) {
		return true
	}
	if node.Type == ast.NodeList || node.Type == ast.NodeItem {
		if node.LastChild != nil && endsWithBlankLine(node.LastChild) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// feed
// ---------------------------------------------------------------------------

func (p *Parser) feed(data []byte, eof bool) {
	repl := []byte{0xEF, 0xBF, 0xBD}
	end := len(data)
	i := 0

	// Skip UTF-8 BOM if present
	if p.lineNum == 0 && p.column == 0 && end >= 3 &&
		data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		i = 3
	} else if p.lastBufferEndedWithCR && i < end && data[i] == '\n' {
		i++
	}

	p.lastBufferEndedWithCR = false
	for i < end {
		eol := i
		process := false
		for eol < end {
			if isLineEndChar(data[eol]) {
				process = true
				break
			}
			if data[eol] == 0 {
				break
			}
			eol++
		}
		if eol >= end && eof {
			process = true
		}

		chunkLen := eol - i
		if process {
			if p.linebuf.Len() > 0 {
				p.linebuf.Append(data[i:eol])
				p.processLine(p.linebuf.Bytes())
				p.linebuf.Clear()
			} else {
				p.processLine(data[i:eol])
			}
		} else {
			if eol < end && data[eol] == 0 {
				p.linebuf.Append(data[i:eol])
				p.linebuf.Append(repl)
			} else {
				p.linebuf.Append(data[i:eol])
			}
		}

		i += chunkLen
		if i < end {
			if data[i] == 0 {
				i++
			} else {
				if data[i] == '\r' {
					i++
					if i >= end {
						p.lastBufferEndedWithCR = true
					}
				}
				if i < end && data[i] == '\n' {
					i++
				}
			}
		}
	}
}

// ---------------------------------------------------------------------------
// processLine
// ---------------------------------------------------------------------------

func (p *Parser) processLine(input []byte) {
	if p.options&ast.OptValidateUTF8 != 0 {
		p.curline.Clear()
		utf8.Check(&p.curline, input)
	} else {
		p.curline.Set(input)
	}

	bytes := p.curline.Len()

	// ensure line ends with a newline
	if bytes == 0 || !isLineEndChar(p.curline.Bytes()[bytes-1]) {
		p.curline.AppendByte('\n')
	}
	bytes = p.curline.Len()

	p.offset = 0
	p.column = 0
	p.firstNonspace = 0
	p.firstNonspaceColumn = 0
	p.thematicBreakKillPos = 0
	p.indent = 0
	p.blank = false
	p.partiallyConsumedTab = false

	data := p.curline.Bytes()
	inputLen := bytes

	p.lineNum++

	lastMatchedContainer, allMatched := p.checkOpenBlocks(data)
	var container *ast.Node

	if lastMatchedContainer != nil {
		container = lastMatchedContainer
		p.openNewBlocks(&container, data, allMatched)
		p.addTextToContainer(container, lastMatchedContainer, data)
	}

	p.lastLineLength = inputLen
	if p.lastLineLength > 0 && data[p.lastLineLength-1] == '\n' {
		p.lastLineLength--
	}
	if p.lastLineLength > 0 && data[p.lastLineLength-1] == '\r' {
		p.lastLineLength--
	}
	p.curline.Clear()
}

// ---------------------------------------------------------------------------
// checkOpenBlocks
// ---------------------------------------------------------------------------

func (p *Parser) checkOpenBlocks(input []byte) (*ast.Node, bool) {
	allMatched := false
	shouldContinue := true
	container := p.root

	for lastChildIsOpen(container) {
		container = container.LastChild
		contType := container.Type

		findFirstNonspace(p, input)

		switch contType {
		case ast.NodeBlockQuote:
			if !p.parseBlockQuotePrefix(input) {
				goto done
			}
		case ast.NodeList:
			if p.blank {
				if (container.Flags&ast.NodeListLastLineBlank) != 0 && p.indent == 0 {
					if p.current.Type == ast.NodeCodeBlock || p.current.Type == ast.NodeHTMLBlock {
						addLine(input, p)
					}
					return nil, false
				}
				container.Flags |= ast.NodeListLastLineBlank
			} else {
				container.Flags &= ^uint16(ast.NodeListLastLineBlank)
			}
		case ast.NodeItem:
			if !p.parseNodeItemPrefix(input, container) {
				goto done
			}
		case ast.NodeCodeBlock:
			if !p.parseCodeBlockPrefix(input, container, &shouldContinue) {
				goto done
			}
		case ast.NodeHeading:
			goto done
		case ast.NodeHTMLBlock:
			if !p.parseHTMLBlockPrefix(container) {
				goto done
			}
		case ast.NodeParagraph:
			if p.blank {
				goto done
			}
		}
	}

	allMatched = true

done:
	if !allMatched {
		container = container.Parent
	}
	if !shouldContinue {
		return nil, false
	}
	return container, allMatched
}

// ---------------------------------------------------------------------------
// openNewBlocks
// ---------------------------------------------------------------------------

func (p *Parser) openNewBlocks(container **ast.Node, input []byte, allMatched bool) {
	maybeLazy := p.current.Type == ast.NodeParagraph
	contType := (*container).Type
	var atxMatch, fenceMatch, htmlMatch, html7Match, setextMatch, listMatch int
	var savePartiallyConsumedTab bool
	var saveOffset, saveColumn int

	for contType != ast.NodeCodeBlock && contType != ast.NodeHTMLBlock {
		findFirstNonspace(p, input)
		indented := p.indent >= codeIndent
		atxMatch = 0
		fenceMatch = 0
		htmlMatch = 0
		html7Match = 0
		setextMatch = 0
		listMatch = 0

		if !indented {
			atxMatch = scanners.ScanATXHeadingStart(input[p.firstNonspace:])
			if atxMatch == 0 {
				fenceMatch = scanners.ScanOpenCodeFence(input[p.firstNonspace:])
			}
			if atxMatch == 0 && fenceMatch == 0 {
				htmlMatch = scanners.ScanHTMLBlockStart(input[p.firstNonspace:])
				if htmlMatch == 0 && contType != ast.NodeParagraph && !maybeLazy {
					html7Match = scanners.ScanHTMLBlockStart7(input[p.firstNonspace:])
				}
			}
			if atxMatch == 0 && fenceMatch == 0 && htmlMatch == 0 && html7Match == 0 && contType == ast.NodeParagraph {
				setextMatch = scanners.ScanSetextHeadingLine(input[p.firstNonspace:])
			}
		}
		if (!indented || contType == ast.NodeList) && p.indent < 4 {
			listMatch = p.parseListMarker(input, p.firstNonspace, contType == ast.NodeParagraph)
		}

		if !indented && peekAt(input, p.firstNonspace) == '>' {
			blockquoteStartPos := p.firstNonspace
			advanceOffset(p, input, p.firstNonspace+1-p.offset, false)
			if isSpaceOrTab(peekAt(input, p.offset)) {
				advanceOffset(p, input, 1, true)
			}
			*container = p.addChild(*container, ast.NodeBlockQuote, blockquoteStartPos+1)
		} else if !indented && atxMatch > 0 {
			headingStartPos := p.firstNonspace
			advanceOffset(p, input, p.firstNonspace+atxMatch-p.offset, false)
			*container = p.addChild(*container, ast.NodeHeading, headingStartPos+1)
			level := 0
			hashPos := bytes.IndexByte(input[p.firstNonspace:], '#')
			if hashPos >= 0 {
				hashPos += p.firstNonspace
				for peekAt(input, hashPos) == '#' {
					level++
					hashPos++
				}
			}
			if (*container).HeadingData == nil {
				(*container).HeadingData = &ast.HeadingData{}
			}
			(*container).HeadingData.Level = int8(level)
			(*container).HeadingData.Setext = false
			(*container).HeadingData.InternalOffset = atxMatch
		} else if !indented && fenceMatch > 0 {
			*container = p.addChild(*container, ast.NodeCodeBlock, p.firstNonspace+1)
			if (*container).CodeData == nil {
				(*container).CodeData = &ast.CodeData{}
			}
			(*container).CodeData.Fenced = true
			(*container).CodeData.FenceChar = input[p.firstNonspace]
			if fenceMatch > 255 {
				fenceMatch = 255
			}
			(*container).CodeData.FenceLength = uint8(fenceMatch)
			(*container).CodeData.FenceOffset = uint8(p.firstNonspace - p.offset)
			(*container).CodeData.Info = ""
			advanceOffset(p, input, p.firstNonspace+fenceMatch-p.offset, false)
		} else if !indented && (htmlMatch > 0 || html7Match > 0) {
			matched := htmlMatch
			if matched == 0 {
				matched = html7Match
			}
			*container = p.addChild(*container, ast.NodeHTMLBlock, p.firstNonspace+1)
			(*container).HTMLBlockType = matched
		} else if !indented && contType == ast.NodeParagraph && setextMatch > 0 {
			hasContent := p.resolveReferenceLinkDefinitions()
			if hasContent {
				(*container).Type = ast.NodeHeading
				if (*container).HeadingData == nil {
					(*container).HeadingData = &ast.HeadingData{}
				}
				(*container).HeadingData.Level = int8(setextMatch)
				(*container).HeadingData.Setext = true
				advanceOffset(p, input, len(input)-1-p.offset, false)
			}
		} else if !indented && !(contType == ast.NodeParagraph && !allMatched) &&
			p.thematicBreakKillPos <= p.firstNonspace &&
			p.scanThematicBreak(input, p.firstNonspace) {
			*container = p.addChild(*container, ast.NodeThematicBreak, p.firstNonspace+1)
			advanceOffset(p, input, len(input)-1-p.offset, false)
		} else if (!indented || contType == ast.NodeList) && p.indent < 4 && listMatch > 0 {
			data := p.parseListMarkerData(input, p.firstNonspace, listMatch)
			advanceOffset(p, input, p.firstNonspace+listMatch-p.offset, false)
			savePartiallyConsumedTab = p.partiallyConsumedTab
			saveOffset = p.offset
			saveColumn = p.column
			for p.column-saveColumn <= 5 && isSpaceOrTab(peekAt(input, p.offset)) {
				advanceOffset(p, input, 1, true)
			}
			i := p.column - saveColumn
			if i >= 5 || i < 1 || isLineEndChar(peekAt(input, p.offset)) {
				data.Padding = listMatch + 1
				p.offset = saveOffset
				p.column = saveColumn
				p.partiallyConsumedTab = savePartiallyConsumedTab
				if i > 0 {
					advanceOffset(p, input, 1, true)
				}
			} else {
				data.Padding = listMatch + i
			}
			data.MarkerOffset = p.indent
			if contType != ast.NodeList || !listsMatch((*container).ListData, data) {
				*container = p.addChild(*container, ast.NodeList, p.firstNonspace+1)
				if (*container).ListData == nil {
					(*container).ListData = &ast.ListData{}
				}
				*(*container).ListData = *data
			}
			*container = p.addChild(*container, ast.NodeItem, p.firstNonspace+1)
			if (*container).ListData == nil {
				(*container).ListData = &ast.ListData{}
			}
			*(*container).ListData = *data
		} else if indented && !maybeLazy && !p.blank {
			advanceOffset(p, input, codeIndent, true)
			*container = p.addChild(*container, ast.NodeCodeBlock, p.offset+1)
			if (*container).CodeData == nil {
				(*container).CodeData = &ast.CodeData{}
			}
			(*container).CodeData.Fenced = false
			(*container).CodeData.FenceChar = 0
			(*container).CodeData.FenceLength = 0
			(*container).CodeData.FenceOffset = 0
			(*container).CodeData.Info = ""
		} else {
			break
		}

		if acceptsLines(*container) {
			break
		}
		contType = (*container).Type
		maybeLazy = false
	}
}

// ---------------------------------------------------------------------------
// addTextToContainer
// ---------------------------------------------------------------------------

func chopTrailingHashtagsFromInput(input []byte) []byte {
	i := len(input)
	for i > 0 && ctype.IsSpace(input[i-1]) {
		i--
	}
	data := input[:i]
	j := len(data)
	origJ := j
	for j > 0 && data[j-1] == '#' {
		j--
	}
	if j != origJ && j > 0 && (data[j-1] == ' ' || data[j-1] == '\t') {
		for j > 0 && (data[j-1] == ' ' || data[j-1] == '\t') {
			j--
		}
		return data[:j]
	}
	return data
}

func (p *Parser) addTextToContainer(container, lastMatchedContainer *ast.Node, input []byte) {
	// Lazy continuation line check
	if p.current != lastMatchedContainer &&
		container == lastMatchedContainer && !p.blank &&
		p.current.Type == ast.NodeParagraph {
		addLine(input, p)
		return
	}

	// Finalize any blocks that were not matched
	for p.current != lastMatchedContainer {
		p.current = p.finalize(p.current)
	}

	findFirstNonspace(p, input)

	if p.blank && container.LastChild != nil {
		setLastLineBlank(container.LastChild, true)
	}

	lastLineBlank := p.blank &&
		container.Type != ast.NodeBlockQuote &&
		container.Type != ast.NodeHeading &&
		container.Type != ast.NodeThematicBreak &&
		!(container.Type == ast.NodeCodeBlock && container.CodeData != nil && container.CodeData.Fenced) &&
		!(container.Type == ast.NodeItem && container.FirstChild == nil && container.StartLine == p.lineNum)
	setLastLineBlank(container, lastLineBlank)

	tmp := container
	for tmp.Parent != nil {
		setLastLineBlank(tmp.Parent, false)
		tmp = tmp.Parent
	}

	if container.Type == ast.NodeCodeBlock {
		addLine(input, p)
	} else if container.Type == ast.NodeHTMLBlock {
		addLine(input, p)
		matchesEndCondition := false
		switch container.HTMLBlockType {
		case 1:
			matchesEndCondition = scanners.ScanHTMLBlockEnd1(input[p.firstNonspace:]) > 0
		case 2:
			matchesEndCondition = scanners.ScanHTMLBlockEnd2(input[p.firstNonspace:]) > 0
		case 3:
			matchesEndCondition = scanners.ScanHTMLBlockEnd3(input[p.firstNonspace:]) > 0
		case 4:
			matchesEndCondition = scanners.ScanHTMLBlockEnd4(input[p.firstNonspace:]) > 0
		case 5:
			matchesEndCondition = scanners.ScanHTMLBlockEnd5(input[p.firstNonspace:]) > 0
		case 6, 7:
			matchesEndCondition = p.blank
		}
		if matchesEndCondition {
			container = p.finalize(container)
		}
	} else if p.blank {
		// do nothing
	} else if acceptsLines(container) {
		if container.Type == ast.NodeHeading && container.HeadingData != nil && !container.HeadingData.Setext {
			input = chopTrailingHashtagsFromInput(input)
		}
		if container.Type == ast.NodeHeading {
			advanceOffset(p, input, p.firstNonspace-p.offset, false)
		}
		addLine(input, p)
	} else {
		container = p.addChild(container, ast.NodeParagraph, p.firstNonspace+1)
		advanceOffset(p, input, p.firstNonspace-p.offset, false)
		addLine(input, p)
	}

	p.current = container
}

// ---------------------------------------------------------------------------
// finalize
// ---------------------------------------------------------------------------

func (p *Parser) finalize(node *ast.Node) *ast.Node {
	parent := node.Parent
	node.Flags &= ^uint16(ast.NodeOpen)

	if p.curline.Len() == 0 {
		node.EndLine = p.lineNum
		node.EndColumn = p.lastLineLength
	} else if node.Type == ast.NodeDocument ||
		(node.Type == ast.NodeCodeBlock && node.CodeData != nil && node.CodeData.Fenced) ||
		(node.Type == ast.NodeHeading && node.HeadingData != nil && node.HeadingData.Setext) {
		node.EndLine = p.lineNum
		node.EndColumn = p.curline.Len()
		if node.EndColumn > 0 {
			b := p.curline.Bytes()
			if b[node.EndColumn-1] == '\n' {
				node.EndColumn--
			}
			if node.EndColumn > 0 && b[node.EndColumn-1] == '\r' {
				node.EndColumn--
			}
		}
	} else {
		node.EndLine = p.lineNum - 1
		node.EndColumn = p.lastLineLength
	}

	if node.Type == ast.NodeParagraph {
		// resolve reference link definitions
		hasContent := p.resolveReferenceLinkDefinitionsInNode(node)
		if !hasContent {
			node.Unlink()
			p.content.Clear()
			return parent
		}
		// strip trailing newlines (C reference behavior)
		for p.content.Len() > 0 && p.content.Bytes()[p.content.Len()-1] == '\n' {
			p.content.Truncate(p.content.Len() - 1)
		}
		node.Data = p.content.String()
		p.content.Clear()
	}

	if node.Type == ast.NodeCodeBlock && node.CodeData != nil {
		if !node.CodeData.Fenced {
			// trim trailing whitespace but stop at newlines (C rtrim behavior)
			data := p.content.Bytes()
			n := len(data)
			for n > 0 && ctype.IsSpace(data[n-1]) && data[n-1] != '\n' {
				n--
			}
			p.content.Truncate(n)
			if p.content.Len() == 0 {
				p.content.AppendByte('\n')
			}
			node.Data = p.content.String()
			p.content.Clear()
		} else {
			buf := p.content.Bytes()
			pos := 0
			for pos < len(buf) && !isLineEndChar(buf[pos]) {
				pos++
			}
			if pos > 0 {
				info := strings.TrimSpace(string(buf[:pos]))
				node.CodeData.Info = html.UnescapeString(info)
			}
			if pos < len(buf) && buf[pos] == '\r' {
				pos++
			}
			if pos < len(buf) && buf[pos] == '\n' {
				pos++
			}
			node.Data = string(buf[pos:])
			p.content.Clear()
		}
	}

	if node.Type == ast.NodeHeading {
		// strip trailing newlines (C reference behavior)
		for p.content.Len() > 0 && p.content.Bytes()[p.content.Len()-1] == '\n' {
			p.content.Truncate(p.content.Len() - 1)
		}
		node.Data = p.content.String()
		p.content.Clear()
	}

	if node.Type == ast.NodeHTMLBlock {
		// strip trailing newlines (C reference behavior)
		for p.content.Len() > 0 && p.content.Bytes()[p.content.Len()-1] == '\n' {
			p.content.Truncate(p.content.Len() - 1)
		}
		node.Data = p.content.String()
		p.content.Clear()
	}

	if node.Type == ast.NodeList {
		node.ListData.Tight = true
		var child *ast.Node
		for child = node.FirstChild; child != nil; child = child.Next {
			if child.Type == ast.NodeItem && child.FirstChild != nil {
				if lastLineBlank(child) {
					node.ListData.Tight = false
					break
				}
				for subchild := child.FirstChild; subchild != nil; subchild = subchild.Next {
					if subchild.Type == ast.NodeHTMLBlock || subchild.Type == ast.NodeCodeBlock ||
						subchild.Type == ast.NodeHeading {
						node.ListData.Tight = false
						break
					}
					if subchild.Type == ast.NodeParagraph && subchild.Next != nil {
						node.ListData.Tight = false
						break
					}
				}
				if !node.ListData.Tight {
					break
				}
			}
		}
	}

	return parent
}

func (p *Parser) finalizeDocument() {
	for p.current != p.root {
		p.current = p.finalize(p.current)
	}
	p.finalize(p.root)
	p.processInlines(p.root)
}

// ---------------------------------------------------------------------------
// addChild
// ---------------------------------------------------------------------------

func (p *Parser) addChild(parent *ast.Node, t ast.NodeType, startColumn int) *ast.Node {
	for !canContainType(parent.Type, t) {
		parent = p.finalize(parent)
	}
	child := ast.NewNode(t)
	child.Flags |= ast.NodeOpen
	child.StartLine = p.lineNum
	child.StartColumn = startColumn
	child.EndLine = p.lineNum
	parent.AppendChild(child)
	return child
}

// ---------------------------------------------------------------------------
// processInlines
// ---------------------------------------------------------------------------

func (p *Parser) processInlines(node *ast.Node) {
	it := ast.NewIter(node)
	for evType := it.Next(); evType != ast.EventDone; evType = it.Next() {
		cur := it.GetNode()
		if evType == ast.EventEnter && containsInlines(cur) {
			ParseInlines(cur, p.refmap, p.options)
		}
	}
}

// ---------------------------------------------------------------------------
// Block prefix parsers
// ---------------------------------------------------------------------------

func (p *Parser) parseBlockQuotePrefix(input []byte) bool {
	if p.indent <= 3 && peekAt(input, p.firstNonspace) == '>' {
		advanceOffset(p, input, p.indent+1, true)
		if isSpaceOrTab(peekAt(input, p.offset)) {
			advanceOffset(p, input, 1, true)
		}
		return true
	}
	return false
}

func (p *Parser) parseNodeItemPrefix(input []byte, container *ast.Node) bool {
	if container.ListData == nil {
		return false
	}
	if p.indent >= container.ListData.MarkerOffset+container.ListData.Padding {
		advanceOffset(p, input, container.ListData.MarkerOffset+container.ListData.Padding, true)
		return true
	}
	if p.blank && container.FirstChild != nil {
		advanceOffset(p, input, p.firstNonspace-p.offset, false)
		return true
	}
	return false
}

func (p *Parser) parseCodeBlockPrefix(input []byte, container *ast.Node, shouldContinue *bool) bool {
	if container.CodeData == nil {
		return false
	}
	if !container.CodeData.Fenced {
		if p.indent >= codeIndent {
			advanceOffset(p, input, codeIndent, true)
			return true
		}
		if p.blank {
			advanceOffset(p, input, p.firstNonspace-p.offset, false)
			return true
		}
	} else {
		matched := 0
		if p.indent <= 3 && peekAt(input, p.firstNonspace) == container.CodeData.FenceChar {
			matched = scanners.ScanCloseCodeFence(input[p.firstNonspace:])
		}
		if matched >= int(container.CodeData.FenceLength) {
			*shouldContinue = false
			advanceOffset(p, input, matched, false)
			p.current = p.finalize(container)
		} else {
			i := int(container.CodeData.FenceOffset)
			for i > 0 && isSpaceOrTab(peekAt(input, p.offset)) {
				advanceOffset(p, input, 1, true)
				i--
			}
			return true
		}
	}
	return false
}

func (p *Parser) parseHTMLBlockPrefix(container *ast.Node) bool {
	if container.HTMLBlockType >= 1 && container.HTMLBlockType <= 5 {
		return true
	}
	if container.HTMLBlockType == 6 || container.HTMLBlockType == 7 {
		return !p.blank
	}
	return false
}

// ---------------------------------------------------------------------------
// thematic break
// ---------------------------------------------------------------------------

func (p *Parser) scanThematicBreak(input []byte, offset int) bool {
	if offset >= len(input) {
		return false
	}
	c := input[offset]
	if c != '*' && c != '-' && c != '_' {
		p.thematicBreakKillPos = offset
		return false
	}
	count := 1
	i := offset
	var nextc byte
	for {
		i++
		if i >= len(input) {
			nextc = 0
			break
		}
		nextc = input[i]
		if nextc == c {
			count++
		} else if nextc != ' ' && nextc != '\t' {
			break
		}
	}
	if count >= 3 && (nextc == '\r' || nextc == '\n') {
		return true
	}
	p.thematicBreakKillPos = i
	return false
}

// ---------------------------------------------------------------------------
// list marker
// ---------------------------------------------------------------------------

func (p *Parser) parseListMarker(input []byte, offset int, inParagraph bool) int {
	if offset >= len(input) {
		return 0
	}
	c := input[offset]
	if c == '*' || c == '-' || c == '+' {
		if offset+1 >= len(input) || isSpaceOrTab(input[offset+1]) {
			return 1
		}
		if offset+1 < len(input) && isLineEndChar(input[offset+1]) {
			if inParagraph {
				return 0
			}
			return 1
		}
		return 0
	}
	if c >= '0' && c <= '9' {
		start := 0
		i := offset
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			start = start*10 + int(input[i]-'0')
			if start > 999999999 {
				return 0
			}
			i++
		}
		if i == offset {
			return 0
		}
		if i < len(input) && (input[i] == '.' || input[i] == ')') {
			if i+1 < len(input) && isLineEndChar(input[i+1]) && inParagraph {
				return 0
			}
			if i+1 >= len(input) || isSpaceOrTab(input[i+1]) || isLineEndChar(input[i+1]) {
				if inParagraph && start != 1 {
					return 0
				}
				return i - offset + 1
			}
		}
	}
	return 0
}

type listMarkerData struct {
	ListType   ast.ListType
	Delimiter  ast.DelimType
	BulletChar byte
	Start      int
	Padding    int
	MarkerOffset int
}

func (p *Parser) parseListMarkerData(input []byte, offset int, matched int) *ast.ListData {
	c := input[offset]
	data := &ast.ListData{}
	if c >= '0' && c <= '9' {
		data.ListType = ast.OrderedList
		start := 0
		i := offset
		for i < len(input) && input[i] >= '0' && input[i] <= '9' {
			start = start*10 + int(input[i]-'0')
			i++
		}
		data.Start = start
		if i < len(input) && input[i] == '.' {
			data.Delimiter = ast.PeriodDelim
		} else {
			data.Delimiter = ast.ParenDelim
		}
	} else {
		data.ListType = ast.BulletList
		data.BulletChar = c
	}
	return data
}

func canContainType(parentType ast.NodeType, childType ast.NodeType) bool {
	switch parentType {
	case ast.NodeDocument, ast.NodeBlockQuote, ast.NodeItem:
		return childType >= ast.NodeDocument && childType <= ast.NodeThematicBreak && childType != ast.NodeItem
	case ast.NodeList:
		return childType == ast.NodeItem
	case ast.NodeParagraph, ast.NodeHeading, ast.NodeEmph, ast.NodeStrong,
		ast.NodeLink, ast.NodeImage, ast.NodeCustomInline:
		return childType >= ast.NodeText && childType <= ast.NodeImage
	default:
		return false
	}
}

func listsMatch(a, b *ast.ListData) bool {
	if a == nil || b == nil {
		return false
	}
	if a.ListType != b.ListType {
		return false
	}
	if a.Delimiter != b.Delimiter {
		return false
	}
	if a.BulletChar != b.BulletChar {
		return false
	}
	return true
}

// ---------------------------------------------------------------------------
// trailing hashtags / reference resolution
// ---------------------------------------------------------------------------


func (p *Parser) resolveReferenceLinkDefinitions() bool {
	return p.resolveReferenceLinkDefinitionsInNode(p.current)
}

func (p *Parser) resolveReferenceLinkDefinitionsInNode(node *ast.Node) bool {
	if node.Type != ast.NodeParagraph {
		return true
	}
	data := p.content.String()
	for {
		pos := ParseReferenceInline([]byte(data), p.refmap)
		if pos == 0 {
			break
		}
		data = data[pos:]
	}
	p.content.Set([]byte(data))
	return p.content.Len() > 0
}

// ---------------------------------------------------------------------------
// Reference inline stub
// ---------------------------------------------------------------------------

func parseReferenceInline(input []byte, refmap *ast.ReferenceMap) int {
	// TODO: implement reference definition parsing
	_ = refmap
	return 0
}
