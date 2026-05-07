// Package commonmark renders a CommonMark AST back to CommonMark text.
package commonmark

import (
	"strings"

	"github.com/briansumma/cmark/internal/render"
	"github.com/briansumma/cmark/pkg/ast"
)

// RenderCommonmark renders the AST as CommonMark text.
func RenderCommonmark(root *ast.Node, options int, width int) string {
	var b strings.Builder
	renderNode(&b, root, options, false)
	return strings.TrimSpace(b.String()) + "\n"
}

func renderNode(b *strings.Builder, node *ast.Node, options int, inTightList bool) {
	switch node.Type {
	case ast.NodeDocument:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options, inTightList)
		}

	case ast.NodeBlockQuote:
		var childContent strings.Builder
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(&childContent, child, options, inTightList)
		}
		for _, line := range strings.Split(childContent.String(), "\n") {
			if line == "" {
				b.WriteString(">\n")
			} else {
				b.WriteString("> ")
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
		b.WriteByte('\n')

	case ast.NodeList:
		tight := false
		if node.ListData != nil {
			tight = node.ListData.Tight
		}
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options, tight)
		}
		if node.Next != nil && node.Next.Type == ast.NodeList {
			b.WriteString("\n<!-- end list -->\n\n")
		}

	case ast.NodeItem:
		markerWidth := 4
		if node.Parent != nil && node.Parent.ListData != nil {
			listData := node.Parent.ListData
			if listData.ListType == ast.BulletList {
				marker := string(listData.BulletChar) + " "
				b.WriteString(marker)
				for i := len(marker); i < markerWidth; i++ {
					b.WriteByte(' ')
				}
			} else {
				listNum := listData.Start
				for tmp := node; tmp != nil; tmp = tmp.Prev {
					if tmp != node {
						listNum++
					}
				}
				delim := "."
				if listData.Delimiter == ast.ParenDelim {
					delim = ")"
				}
				marker := render.Itoa(listNum) + delim + " "
				b.WriteString(marker)
				for i := len(marker); i < markerWidth; i++ {
					b.WriteByte(' ')
				}
			}
		}
		if node.FirstChild == nil {
			b.WriteByte('\n')
		} else {
			var itemContent strings.Builder
			for child := node.FirstChild; child != nil; child = child.Next {
				renderNode(&itemContent, child, options, inTightList)
			}
			lines := strings.Split(itemContent.String(), "\n")
			first := true
			for _, line := range lines {
				if first {
					first = false
					b.WriteString(line)
					b.WriteByte('\n')
				} else if line != "" {
					b.WriteString(strings.Repeat(" ", markerWidth))
					b.WriteString(line)
					b.WriteByte('\n')
				} else if !inTightList {
					b.WriteByte('\n')
				}
			}
		}
		if !inTightList {
			b.WriteByte('\n')
		}

	case ast.NodeHeading:
		if node.HeadingData != nil && node.HeadingData.Setext && node.HeadingData.Level <= 2 {
			var headingText strings.Builder
			for child := node.FirstChild; child != nil; child = child.Next {
				renderInline(&headingText, child, options)
			}
			b.WriteString(headingText.String())
			b.WriteByte('\n')
			if node.HeadingData.Level == 1 {
				b.WriteString("===\n")
			} else {
				b.WriteString("---\n")
			}
		} else {
			level := 1
			if node.HeadingData != nil {
				level = int(node.HeadingData.Level)
			}
			for i := 0; i < level; i++ {
				b.WriteByte('#')
			}
			b.WriteByte(' ')
			for child := node.FirstChild; child != nil; child = child.Next {
				renderInline(b, child, options)
			}
			b.WriteByte('\n')
		}
		b.WriteByte('\n')

	case ast.NodeCodeBlock:
		if node.CodeData != nil && node.CodeData.Fenced {
			info := ""
			if node.CodeData != nil {
				info = node.CodeData.Info
			}
			fenceChar := '`'
			if strings.ContainsRune(info, '`') {
				fenceChar = '~'
			}
			numticks := longestBacktickSequence(node.Data) + 1
			if numticks < 3 {
				numticks = 3
			}
			for i := 0; i < numticks; i++ {
				b.WriteRune(rune(fenceChar))
			}
			if info != "" {
				b.WriteByte(' ')
				b.WriteString(info)
			}
			b.WriteByte('\n')
			b.WriteString(node.Data)
			if node.Data != "" && !strings.HasSuffix(node.Data, "\n") {
				b.WriteByte('\n')
			}
			for i := 0; i < numticks; i++ {
				b.WriteRune(rune(fenceChar))
			}
			b.WriteByte('\n')
		} else {
			for _, line := range strings.Split(node.Data, "\n") {
				b.WriteString("    ")
				b.WriteString(line)
				b.WriteByte('\n')
			}
		}
		b.WriteByte('\n')

	case ast.NodeHTMLBlock:
		b.WriteString(node.Data)
		b.WriteByte('\n')
		b.WriteByte('\n')

	case ast.NodeThematicBreak:
		b.WriteString("-----\n")
		b.WriteByte('\n')

	case ast.NodeParagraph:
		if inTightList {
			for child := node.FirstChild; child != nil; child = child.Next {
				renderInline(b, child, options)
			}
		} else {
			for child := node.FirstChild; child != nil; child = child.Next {
				renderInline(b, child, options)
			}
			b.WriteByte('\n')
		}
		if !inTightList {
			b.WriteByte('\n')
		}
	}
}

func renderInline(b *strings.Builder, node *ast.Node, options int) {
	switch node.Type {
	case ast.NodeText:
		b.WriteString(escapeCommonMark(node.Data))
	case ast.NodeSoftbreak:
		b.WriteByte('\n')
	case ast.NodeLinebreak:
		b.WriteString("  \n")
	case ast.NodeCode:
		code := node.Data
		numticks := shortestUnusedBacktickSequence(code)
		extraSpaces := code == "" || strings.HasPrefix(code, "`") || strings.HasSuffix(code, "`")
		if !extraSpaces {
			trimmed := strings.TrimSpace(code)
			if trimmed != "" && (strings.HasPrefix(code, " ") || strings.HasSuffix(code, " ")) {
				extraSpaces = true
			}
		}
		for i := 0; i < numticks; i++ {
			b.WriteByte('`')
		}
		if extraSpaces {
			b.WriteByte(' ')
		}
		b.WriteString(code)
		if extraSpaces {
			b.WriteByte(' ')
		}
		for i := 0; i < numticks; i++ {
			b.WriteByte('`')
		}
	case ast.NodeEmph:
		b.WriteByte('*')
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteByte('*')
	case ast.NodeStrong:
		b.WriteString("**")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("**")
	case ast.NodeLink:
		b.WriteByte('[')
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("](")
		if node.LinkData != nil {
			b.WriteString(escapeURL(node.LinkData.URL))
		}
		if node.LinkData != nil && node.LinkData.Title != "" {
			b.WriteString(" \"")
			b.WriteString(escapeTitle(node.LinkData.Title))
			b.WriteString("\"")
		}
		b.WriteByte(')')
	case ast.NodeImage:
		b.WriteString("![")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("](")
		if node.LinkData != nil {
			b.WriteString(escapeURL(node.LinkData.URL))
		}
		if node.LinkData != nil && node.LinkData.Title != "" {
			b.WriteString(" \"")
			b.WriteString(escapeTitle(node.LinkData.Title))
			b.WriteString("\"")
		}
		b.WriteByte(')')
	case ast.NodeHTMLInline:
		b.WriteString(node.Data)
	}
}

func escapeCommonMark(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '*', '_', '[', ']', '#', '<', '>', '`', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '!':
			if i+1 < len(s) && s[i+1] == '[' {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
		case '&':
			if i+1 < len(s) && isAlpha(s[i+1]) {
				b.WriteByte('\\')
			}
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func escapeURL(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case ' ':
			b.WriteString("%20")
		case '`', '<', '>', '\\', ')', '(':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func escapeTitle(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func longestBacktickSequence(s string) int {
	longest := 0
	current := 0
	for i := 0; i <= len(s); i++ {
		if i < len(s) && s[i] == '`' {
			current++
		} else {
			if current > longest {
				longest = current
			}
			current = 0
		}
	}
	return longest
}

func shortestUnusedBacktickSequence(s string) int {
	used := uint32(1)
	current := 0
	for i := 0; i <= len(s); i++ {
		if i < len(s) && s[i] == '`' {
			current++
		} else {
			if current > 0 && current < 32 {
				used |= (1 << current)
			}
			current = 0
		}
	}
	i := 0
	for i < 32 && used&1 != 0 {
		used >>= 1
		i++
	}
	return i
}

