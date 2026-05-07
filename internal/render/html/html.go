package html

import (
	"strings"

	"github.com/briansumma/cmark/internal/render"
	"github.com/briansumma/cmark/pkg/ast"
)

func escapeHTML(s string) string {
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

// RenderHTML renders the AST as an HTML fragment.
func RenderHTML(root *ast.Node, options int) string {
	var b strings.Builder
	renderNode(&b, root, options, false)
	return b.String()
}

func renderNode(b *strings.Builder, node *ast.Node, options int, inTightList bool) {
	switch node.Type {
	case ast.NodeDocument:
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options, inTightList)
		}

	case ast.NodeBlockQuote:
		b.WriteString("<blockquote>\n")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options, inTightList)
		}
		b.WriteString("</blockquote>\n")

	case ast.NodeList:
		tight := false
		if node.ListData != nil {
			tight = node.ListData.Tight
		}
		if node.ListData != nil && node.ListData.ListType == ast.OrderedList {
			start := ""
			if node.ListData.Start != 0 && node.ListData.Start != 1 {
				start = " start=\"" + render.Itoa(node.ListData.Start) + "\""
			}
			b.WriteString("<ol" + start + ">\n")
		} else {
			b.WriteString("<ul>\n")
		}
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options, tight)
		}
		if node.ListData != nil && node.ListData.ListType == ast.OrderedList {
			b.WriteString("</ol>\n")
		} else {
			b.WriteString("</ul>\n")
		}

	case ast.NodeItem:
		b.WriteString("<li>")
		if !inTightList {
			b.WriteString("\n")
		}
		for child := node.FirstChild; child != nil; child = child.Next {
			renderNode(b, child, options, inTightList)
		}
		b.WriteString("</li>\n")

	case ast.NodeCodeBlock:
		info := ""
		if node.CodeData != nil {
			info = node.CodeData.Info
		}
		b.WriteString("<pre><code")
		if info != "" {
			lang := strings.SplitN(info, " ", 2)[0]
			b.WriteString(" class=\"language-")
			b.WriteString(escapeHTML(lang))
			b.WriteString("\"")
		}
		b.WriteString(">")
		b.WriteString(escapeHTML(node.Data))
		b.WriteString("</code></pre>\n")

	case ast.NodeParagraph:
		if inTightList {
			for child := node.FirstChild; child != nil; child = child.Next {
				renderInline(b, child, options)
			}
		} else {
			b.WriteString("<p>")
			for child := node.FirstChild; child != nil; child = child.Next {
				renderInline(b, child, options)
			}
			b.WriteString("</p>\n")
		}

	case ast.NodeHeading:
		level := 1
		if node.HeadingData != nil {
			level = int(node.HeadingData.Level)
		}
		b.WriteString("<h")
		b.WriteByte('0' + byte(level))
		b.WriteString(">")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("</h")
		b.WriteByte('0' + byte(level))
		b.WriteString(">\n")

	case ast.NodeThematicBreak:
		b.WriteString("<hr />\n")

	case ast.NodeHTMLBlock:
		b.WriteString(node.Data)
		b.WriteString("\n")
	}
}

func renderInline(b *strings.Builder, node *ast.Node, options int) {
	switch node.Type {
	case ast.NodeText:
		b.WriteString(escapeHTML(node.Data))
	case ast.NodeSoftbreak:
		b.WriteString("\n")
	case ast.NodeLinebreak:
		b.WriteString("<br />\n")
	case ast.NodeCode:
		b.WriteString("<code>")
		b.WriteString(escapeHTML(node.Data))
		b.WriteString("</code>")
	case ast.NodeEmph:
		b.WriteString("<em>")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("</em>")
	case ast.NodeStrong:
		b.WriteString("<strong>")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("</strong>")
	case ast.NodeLink:
		b.WriteString("<a href=\"")
		if node.LinkData != nil {
			b.WriteString(escapeHTML(node.LinkData.URL))
		}
		b.WriteString("\"")
		if node.LinkData != nil && node.LinkData.Title != "" {
			b.WriteString(" title=\"")
			b.WriteString(escapeHTML(node.LinkData.Title))
			b.WriteString("\"")
		}
		b.WriteString(">")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("</a>")
	case ast.NodeImage:
		b.WriteString("<img src=\"")
		if node.LinkData != nil {
			b.WriteString(escapeHTML(node.LinkData.URL))
		}
		b.WriteString("\" alt=\"")
		for child := node.FirstChild; child != nil; child = child.Next {
			renderInline(b, child, options)
		}
		b.WriteString("\"")
		if node.LinkData != nil && node.LinkData.Title != "" {
			b.WriteString(" title=\"")
			b.WriteString(escapeHTML(node.LinkData.Title))
			b.WriteString("\"")
		}
		b.WriteString(" />")
	case ast.NodeHTMLInline:
		b.WriteString(node.Data)
	}
}

