package man

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
	got := RenderMan(root, 0, 0)
	if got == "" {
		t.Fatal("renderer produced empty output")
	}
	if !strings.Contains(got, ".TH") {
		t.Fatal("missing .TH")
	}
	if !strings.Contains(got, ".SH") {
		t.Fatal("missing .SH")
	}
}
