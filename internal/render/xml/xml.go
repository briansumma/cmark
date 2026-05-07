// Package xml renders a CommonMark AST as XML.
package xml

import (
	"strings"

	"github.com/briansumma/cmark/internal/render"
	"github.com/briansumma/cmark/pkg/ast"
)

func escapeXML(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// RenderXML renders the AST as XML.
func RenderXML(root *ast.Node, options int) string {
	var b strings.Builder
	b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	b.WriteString("<!DOCTYPE document SYSTEM \"CommonMark.dtd\">\n")
	renderIter(&b, root, options)
	return b.String()
}

func renderIter(b *strings.Builder, root *ast.Node, options int) {
	iter := ast.NewIter(root)
	indent := 0
	for ev := iter.Next(); ev != ast.EventDone; ev = iter.Next() {
		node := iter.GetNode()
		if ev == ast.EventEnter {
			literal := false
			writeIndent(b, indent)
			b.WriteByte('<')
			b.WriteString(ast.NodeTypeString(node.Type))

			switch node.Type {
			case ast.NodeDocument:
				b.WriteString(" xmlns=\"http://commonmark.org/xml/1.0\"")
			case ast.NodeText, ast.NodeCode, ast.NodeHTMLBlock, ast.NodeHTMLInline:
				b.WriteString(" xml:space=\"preserve\">")
				b.WriteString(escapeXML(node.Data))
				b.WriteString("</")
				b.WriteString(ast.NodeTypeString(node.Type))
				literal = true
			case ast.NodeList:
				if node.ListData != nil {
					if node.ListData.ListType == ast.OrderedList {
						b.WriteString(" type=\"ordered\"")
						b.WriteString(" start=\"")
						b.WriteString(render.Itoa(node.ListData.Start))
						b.WriteString("\"")
						if node.ListData.Delimiter == ast.ParenDelim {
							b.WriteString(" delimiter=\"paren\"")
						} else if node.ListData.Delimiter == ast.PeriodDelim {
							b.WriteString(" delimiter=\"period\"")
						}
					} else {
						b.WriteString(" type=\"bullet\"")
					}
					if node.ListData.Tight {
						b.WriteString(" tight=\"true\"")
					} else {
						b.WriteString(" tight=\"false\"")
					}
				}
			case ast.NodeHeading:
				if node.HeadingData != nil {
					b.WriteString(" level=\"")
					b.WriteString(render.Itoa(int(node.HeadingData.Level)))
					b.WriteString("\"")
				}
			case ast.NodeCodeBlock:
				if node.CodeData != nil && node.CodeData.Info != "" {
					b.WriteString(" info=\"")
					b.WriteString(escapeXML(node.CodeData.Info))
					b.WriteString("\"")
				}
				b.WriteString(" xml:space=\"preserve\">")
				b.WriteString(escapeXML(node.Data))
				b.WriteString("</")
				b.WriteString(ast.NodeTypeString(node.Type))
				literal = true
			case ast.NodeLink, ast.NodeImage:
				if node.LinkData != nil {
					b.WriteString(" destination=\"")
					b.WriteString(escapeXML(node.LinkData.URL))
					b.WriteString("\"")
					if node.LinkData.Title != "" {
						b.WriteString(" title=\"")
						b.WriteString(escapeXML(node.LinkData.Title))
						b.WriteString("\"")
					}
				}
			}

			if node.FirstChild != nil {
				indent += 2
			} else if !literal {
				b.WriteString(" /")
			}
			b.WriteString(">\n")
		} else {
			if node.FirstChild != nil {
				indent -= 2
				writeIndent(b, indent)
				b.WriteString("</")
				b.WriteString(ast.NodeTypeString(node.Type))
				b.WriteString(">\n")
			}
		}
	}
}

func writeIndent(b *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		b.WriteByte(' ')
	}
}

