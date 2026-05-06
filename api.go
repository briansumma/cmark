// Package cmark implements CommonMark parsing, manipulating, and rendering.
//
// This is a pure-Go port of the cmark C library.
package cmark

import (
	"io"

	"github.com/briansumma/cmark/internal/parser"
	"github.com/briansumma/cmark/internal/render/commonmark"
	"github.com/briansumma/cmark/internal/render/html"
	"github.com/briansumma/cmark/internal/render/latex"
	"github.com/briansumma/cmark/internal/render/man"
	"github.com/briansumma/cmark/internal/render/xml"
	"github.com/briansumma/cmark/pkg/ast"
)

// Version returns the library version as an integer.
func Version() int {
	return 0x001F02 // 0.31.2
}

// VersionString returns the library version string.
func VersionString() string {
	return "0.31.2"
}

// MarkdownToHTML converts a CommonMark document to HTML.
func MarkdownToHTML(text []byte, options ast.Options) (string, error) {
	root, err := parser.ParseDocument(text, options)
	if err != nil {
		return "", err
	}
	return html.RenderHTML(root, int(options)), nil
}

// ParseDocument parses a CommonMark document from a byte slice.
func ParseDocument(data []byte, options ast.Options) (*ast.Node, error) {
	return parser.ParseDocument(data, options)
}

// ParseFile parses a CommonMark document from a file.
func ParseFile(r io.Reader, options ast.Options) (*ast.Node, error) {
	return parser.ParseFile(r, options)
}

// RenderXML renders a node tree as XML.
func RenderXML(root *ast.Node, options ast.Options) string {
	return xml.RenderXML(root, int(options))
}

// RenderHTML renders a node tree as HTML.
func RenderHTML(root *ast.Node, options ast.Options) string {
	return html.RenderHTML(root, int(options))
}

// RenderMan renders a node tree as a groff man page.
func RenderMan(root *ast.Node, options ast.Options, width int) string {
	return man.RenderMan(root, int(options), width)
}

// RenderCommonmark renders a node tree as CommonMark text.
func RenderCommonmark(root *ast.Node, options ast.Options, width int) string {
	return commonmark.RenderCommonmark(root, int(options), width)
}

// RenderLatex renders a node tree as LaTeX.
func RenderLatex(root *ast.Node, options ast.Options, width int) string {
	return latex.RenderLatex(root, int(options), width)
}
