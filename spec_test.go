package cmark

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/briansumma/cmark/internal/parser"
	"github.com/briansumma/cmark/internal/render/html"
	"github.com/briansumma/cmark/pkg/ast"
)

type specTest struct {
	Markdown string `json:"markdown"`
	HTML     string `json:"html"`
	Example  int    `json:"example"`
	Section  string `json:"section"`
}

func TestSpec(t *testing.T) {
	data, err := os.ReadFile("spec.json")
	if err != nil {
		t.Skipf("spec.json not found: %v", err)
	}
	var tests []specTest
	if err := json.Unmarshal(data, &tests); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, tc := range tests {
		tc := tc
		t.Run(fmt.Sprintf("Example_%d", tc.Example), func(t *testing.T) {
			root, err := parser.ParseDocument([]byte(tc.Markdown), ast.OptDefault)
			if err != nil {
				t.Fatalf("parse error: %v", err)
			}
			got := html.RenderHTML(root, int(ast.OptDefault))
			if got != tc.HTML {
				t.Errorf("Example %d:\ninput:\n%s\nwant:\n%s\ngot:\n%s",
					tc.Example, tc.Markdown, tc.HTML, got)
			}
		})
	}
}
