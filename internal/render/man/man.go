// Package man renders a CommonMark AST as a groff man page.
package man

import (
	"strings"

	"github.com/briansumma/cmark/internal/render"
	"github.com/briansumma/cmark/pkg/ast"
)

// RenderMan renders the AST as a groff man page fragment.
func RenderMan(root *ast.Node, options int, width int) string {
	var b strings.Builder
	b.WriteString(".TH \"\" \"\" \"\" \"\" \"\"\n")
	renderNode(&b, root, options)
	return b.String()
}

func renderNode(b *strings.Builder, node *ast.Node, options int) {
	switch node.Type {
	case ast.NodeDocument:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}

	case ast.NodeBlockQuote:
		b.WriteString(".RS\n")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}
		b.WriteString(".RE\n")

	case ast.NodeList:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}

	case ast.NodeItem:
		if node.Parent != nil && node.Parent.ListData != nil {
			listData := node.Parent.ListData
			if listData.ListType == ast.BulletList {
				b.WriteString(".IP \"\\[bu]\" 2\n")
			} else {
				listNum := listData.Start
				for tmp := node; tmp != nil; tmp = tmp.Prev {
					if tmp != node {
						listNum++
					}
				}
				marker := render.Itoa(listNum) + "."
				b.WriteString(".IP \"")
				b.WriteString(marker)
				b.WriteString("\" 2\n")
			}
		}
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}
		b.WriteByte('\n')

	case ast.NodeHeading:
		if node.HeadingData != nil && node.HeadingData.Level == 1 {
			b.WriteString(".SH \"")
		} else {
			b.WriteString(".SS \"")
		}
		var headingText strings.Builder
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(&headingText, child, options)
		}
		b.WriteString(escapeMan(headingText.String()))
		b.WriteString("\"\n")

	case ast.NodeCodeBlock:
		b.WriteString(".IP \"\" 2\n")
		b.WriteString(".nf\n")
		for _, line := range strings.Split(node.Data, "\n") {
			b.WriteString(line)
			b.WriteByte('\n')
		}
		b.WriteString(".fi\n")

	case ast.NodeHTMLBlock:
		// omitted

	case ast.NodeThematicBreak:
		b.WriteString(".PP\n")
		b.WriteString("* * *\n")

	case ast.NodeParagraph:
		if node.Parent != nil && node.Parent.Type == ast.NodeItem && node.Prev == nil {
			// no .PP for first paragraph in list item
		} else {
			b.WriteString(".PP\n")
		}
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteByte('\n')
	}
}

func renderInline(b *strings.Builder, node *ast.Node, options int) {
	switch node.Type {
	case ast.NodeText:
		b.WriteString(escapeMan(node.Data))
	case ast.NodeSoftbreak:
		b.WriteByte('\n')
	case ast.NodeLinebreak:
		b.WriteString(".br\n")
	case ast.NodeCode:
		b.WriteString("\\fB")
		b.WriteString(escapeMan(node.Data))
		b.WriteString("\\fR")
	case ast.NodeEmph:
		b.WriteString("\\fI")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("\\fR")
	case ast.NodeStrong:
		b.WriteString("\\fB")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("\\fR")
	case ast.NodeLink:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
	case ast.NodeImage:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
	case ast.NodeHTMLInline:
		// omitted
	}
}

func escapeMan(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			b.WriteString("\\e")
		case '-':
			b.WriteString("\\-")
		case '.':
			if i == 0 {
				b.WriteString("\\&.")
			} else {
				b.WriteByte(c)
			}
		case '\'':
			if i == 0 {
				b.WriteString("\\&'")
			} else {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

