// Package render provides the generic renderer infrastructure.
package render

import (
	"github.com/briansumma/cmark/internal/buffer"
	"github.com/briansumma/cmark/pkg/ast"
)

// Escaping mode for output.
type Escaping int

const (
	Literal Escaping = iota
	Normal
	Title
	URL
)

// Renderer holds state during rendering.
type Renderer struct {
	Options         int
	Buffer          *buffer.Buffer
	Prefix          *buffer.Buffer
	Column          int
	Width           int
	NeedCR          int
	LastBreakable   int
	BeginLine       bool
	BeginContent    bool
	NoLineBreaks    bool
	InTightListItem bool
	Outc            func(r *Renderer, esc Escaping, c rune, source byte)
	CR              func(r *Renderer)
	BlankLine       func(r *Renderer)
	Out             func(r *Renderer, s string, wrap bool, esc Escaping)
}

// RenderASCII renders an ASCII string.
func RenderASCII(r *Renderer, s string) {
	// TODO: stub
}

// RenderCodePoint renders a single Unicode code point.
func RenderCodePoint(r *Renderer, c uint32) {
	// TODO: stub
}

// Render walks the AST and renders it using the provided callbacks.
func Render(root *ast.Node, options int, width int,
	outc func(*Renderer, Escaping, rune, byte),
	renderNode func(*Renderer, *ast.Node, ast.EventType, int) int) string {
	// TODO: stub
	return ""
}

// Itoa converts an integer to its decimal string representation.
func Itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [32]byte
	i := len(digits) - 1
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		digits[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	if neg {
		digits[i] = '-'
		i--
	}
	return string(digits[i+1:])
}
