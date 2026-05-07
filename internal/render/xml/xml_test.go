package xml

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
	got := RenderXML(root, 0)
	if got == "" {
		t.Fatal("renderer produced empty output")
	}
	if !strings.Contains(got, "<document") {
		t.Fatal("missing document tag")
	}
	if !strings.Contains(got, "<heading level=\"1\"") {
		t.Fatal("missing heading tag")
	}
	if !strings.Contains(got, "<paragraph>") {
		t.Fatal("missing paragraph tag")
	}
}
