package latex

import (
	"strings"
	"testing"

	"github.com/briansumma/cmark/internal/parser"
)

func TestRenderer(t *testing.T) {
	input := []byte("# Hello\n\nworld.\n")
	root, err := parser.ParseDocument(input, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := RenderLatex(root, 0, 0)
	if got == "" {
		t.Fatal("renderer produced empty output")
	}
	if !strings.Contains(got, "\\documentclass") {
		t.Fatal("missing documentclass")
	}
	if !strings.Contains(got, "\\section*{") {
		t.Fatal("missing section")
	}
}
