// Package latex renders a CommonMark AST as LaTeX.
package latex

import (
	"strings"

	"github.com/briansumma/cmark/pkg/ast"
)

// RenderLatex renders the AST as LaTeX.
func RenderLatex(root *ast.Node, options int, width int) string {
	var b strings.Builder
	b.WriteString("\\documentclass{article}\n")
	b.WriteString("\\begin{document}\n")
	renderNode(&b, root, options)
	b.WriteString("\\end{document}\n")
	return b.String()
}

func renderNode(b *strings.Builder, node *ast.Node, options int) {
	switch node.Type {
	case ast.NodeDocument:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}

	case ast.NodeBlockQuote:
		b.WriteString("\\begin{quote}\n")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}
		b.WriteString("\\end{quote}\n")

	case ast.NodeList:
		if node.ListData != nil && node.ListData.ListType == ast.OrderedList {
			b.WriteString("\\begin{enumerate}\n")
			for child := node.FirstChild; child != nil; child = child.Next {
				renderNode(b, child, options)
			}
			b.WriteString("\\end{enumerate}\n")
		} else {
			b.WriteString("\\begin{itemize}\n")
			for child := node.FirstChild; child != nil; child = child.Next {
				renderNode(b, child, options)
			}
			b.WriteString("\\end{itemize}\n")
		}

	case ast.NodeItem:
		b.WriteString("\\item ")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options)
		}
		b.WriteByte('\n')

	case ast.NodeHeading:
		level := 1
		if node.HeadingData != nil {
			level = int(node.HeadingData.Level)
		}
		switch level {
		case 1:
			b.WriteString("\\section*{")
		case 2:
			b.WriteString("\\subsection*{")
		case 3:
			b.WriteString("\\subsubsection*{")
		default:
			b.WriteString("\\paragraph*{")
		}
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("}\n")

	case ast.NodeCodeBlock:
		b.WriteString("\\begin{verbatim}\n")
		b.WriteString(node.Data)
		if node.Data != "" && !strings.HasSuffix(node.Data, "\n") {
			b.WriteByte('\n')
		}
		b.WriteString("\\end{verbatim}\n")

	case ast.NodeHTMLBlock:
		// omitted

	case ast.NodeThematicBreak:
		b.WriteString("\\begin{center}\\rule{0.5\\linewidth}{\\linethickness}\\end{center}\n")

	case ast.NodeParagraph:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteByte('\n')
	}
}

func renderInline(b *strings.Builder, node *ast.Node, options int) {
	switch node.Type {
	case ast.NodeText:
		b.WriteString(escapeLatex(node.Data))
	case ast.NodeSoftbreak:
		b.WriteByte('\n')
	case ast.NodeLinebreak:
		b.WriteString("\\\\\n")
	case ast.NodeCode:
		b.WriteString("\\texttt{")
		b.WriteString(escapeLatex(node.Data))
		b.WriteString("}")
	case ast.NodeEmph:
		b.WriteString("\\emph{")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("}")
	case ast.NodeStrong:
		b.WriteString("\\textbf{")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("}")
	case ast.NodeLink:
		b.WriteString("\\href{")
		if node.LinkData != nil {
			b.WriteString(escapeLatex(node.LinkData.URL))
		}
		b.WriteString("}{")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("}")
	case ast.NodeImage:
		b.WriteString("\\includegraphics{")
		if node.LinkData != nil {
			b.WriteString(escapeLatex(node.LinkData.URL))
		}
		b.WriteString("}")
	case ast.NodeHTMLInline:
		b.WriteString(node.Data)
	}
}

func escapeLatex(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '\\':
			b.WriteString("\\textbackslash{}")
		case '{', '}', '#', '%', '&':
			b.WriteByte('\\')
			b.WriteByte(c)
		case '$':
			b.WriteString("\\$")
		case '_':
			b.WriteString("\\_")
		case '~':
			b.WriteString("\\textasciitilde{}")
		case '^':
			b.WriteString("\\^{}")
		case '-':
			if i+1 < len(s) && s[i+1] == '-' {
				b.WriteString("-{}")
			} else {
				b.WriteByte(c)
			}
		case '|':
			b.WriteString("\\textbar{}")
		case '<':
			b.WriteString("\\textless{}")
		case '>':
			b.WriteString("\\textgreater{}")
		case 0xC2:
			// possible UTF-8 smart quotes/dashes
			if i+1 < len(s) {
				switch s[i+1] {
				case 0xA0:
					b.WriteByte('~')
					i++
				case 0xA6:
					b.WriteString("\\ldots{}")
					i++
				default:
					b.WriteByte(c)
				}
			} else {
				b.WriteByte(c)
			}
		case 0xE2:
			if i+2 < len(s) {
				switch {
				case s[i+1] == 0x80 && s[i+2] == 0x98:
					b.WriteByte('`')
					i += 2
				case s[i+1] == 0x80 && s[i+2] == 0x99:
					b.WriteByte('\'')
					i += 2
				case s[i+1] == 0x80 && s[i+2] == 0x9C:
					b.WriteString("``")
					i += 2
				case s[i+1] == 0x80 && s[i+2] == 0x9D:
					b.WriteString("''")
					i += 2
				case s[i+1] == 0x80 && s[i+2] == 0x93:
					b.WriteString("--")
					i += 2
				case s[i+1] == 0x80 && s[i+2] == 0x94:
					b.WriteString("---")
					i += 2
				default:
					b.WriteByte(c)
				}
			} else {
				b.WriteByte(c)
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
