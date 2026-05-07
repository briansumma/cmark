package parser

import (
	"testing"

	"github.com/briansumma/cmark/pkg/ast"
)

func TestParse(t *testing.T) {
	data := []byte("# Hello\n\nworld.\n")
	root, err := ParseDocument(data, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root == nil {
		t.Fatal("expected non-nil root")
	}
}

func TestListTwoItems(t *testing.T) {
	input := []byte("* item 1\n* item 2\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeList {
		t.Fatalf("expected single list node, got %v", root.FirstChild)
	}
	list := root.FirstChild
	if list.FirstChild == nil || list.FirstChild.Type != ast.NodeItem {
		t.Fatal("expected first child to be item")
	}
	if list.FirstChild.Next == nil || list.FirstChild.Next.Type != ast.NodeItem {
		t.Fatal("expected second item")
	}
	if list.FirstChild.Next.Next != nil {
		t.Fatal("expected exactly two items")
	}
}

func TestOrderedList(t *testing.T) {
	input := []byte("1. first\n2. second\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeList {
		t.Fatalf("expected single list node, got %v", root.FirstChild)
	}
	list := root.FirstChild
	if list.ListData == nil || list.ListData.ListType != ast.OrderedList {
		t.Fatal("expected ordered list")
	}
	if list.FirstChild == nil || list.FirstChild.Type != ast.NodeItem {
		t.Fatal("expected first child to be item")
	}
	if list.FirstChild.Next == nil || list.FirstChild.Next.Type != ast.NodeItem {
		t.Fatal("expected second item")
	}
	if list.FirstChild.Next.Next != nil {
		t.Fatal("expected exactly two items")
	}
}

func TestNestedList(t *testing.T) {
	input := []byte("* outer\n  * inner\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeList {
		t.Fatalf("expected list node at root, got %v", root.FirstChild)
	}
	outerList := root.FirstChild
	if outerList.FirstChild == nil || outerList.FirstChild.Type != ast.NodeItem {
		t.Fatal("expected outer item")
	}
	outerItem := outerList.FirstChild
	// The inner list should be inside the outer item
	if outerItem.FirstChild == nil {
		t.Fatal("expected inner content inside outer item")
	}
	// inner content may be paragraph + list, or just list
	foundInnerList := false
	for child := outerItem.FirstChild; child != nil; child = child.Next {
		if child.Type == ast.NodeList {
			foundInnerList = true
			if child.ListData == nil || child.ListData.ListType != ast.BulletList {
				t.Fatal("expected inner bullet list")
			}
			if child.FirstChild == nil || child.FirstChild.Type != ast.NodeItem {
				t.Fatal("expected inner list item")
			}
		}
	}
	if !foundInnerList {
		t.Fatal("expected nested list inside outer item")
	}
}

func TestEmphasis(t *testing.T) {
	input := []byte("*hello*\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeEmph {
		t.Fatalf("expected emph, got %v", p.FirstChild)
	}
	emph := p.FirstChild
	if emph.FirstChild == nil || emph.FirstChild.Type != ast.NodeText || emph.FirstChild.Data != "hello" {
		t.Fatalf("expected text 'hello' inside emph, got %v", emph.FirstChild)
	}
}

func TestStrong(t *testing.T) {
	input := []byte("**hello**\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeStrong {
		t.Fatalf("expected strong, got %v", p.FirstChild)
	}
	strong := p.FirstChild
	if strong.FirstChild == nil || strong.FirstChild.Type != ast.NodeText || strong.FirstChild.Data != "hello" {
		t.Fatalf("expected text 'hello' inside strong, got %v", strong.FirstChild)
	}
}

func TestCodeSpan(t *testing.T) {
	input := []byte("`code`\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeCode {
		t.Fatalf("expected code, got %v", p.FirstChild)
	}
	code := p.FirstChild
	if code.Data != "code" {
		t.Fatalf("expected code data 'code', got %q", code.Data)
	}
}

func TestLink(t *testing.T) {
	input := []byte("[text](url)\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeLink {
		t.Fatalf("expected link, got %v", p.FirstChild)
	}
	link := p.FirstChild
	if link.LinkData == nil || link.LinkData.URL != "url" {
		t.Fatalf("expected link url 'url', got %v", link.LinkData)
	}
	if link.FirstChild == nil || link.FirstChild.Type != ast.NodeText || link.FirstChild.Data != "text" {
		t.Fatalf("expected text 'text' inside link, got %v", link.FirstChild)
	}
}

func TestAutolink(t *testing.T) {
	input := []byte("<http://example.com>\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeLink {
		t.Fatalf("expected link, got %v", p.FirstChild)
	}
	link := p.FirstChild
	if link.LinkData == nil || link.LinkData.URL != "http://example.com" {
		t.Fatalf("expected autolink url, got %v", link.LinkData)
	}
}

func TestEntity(t *testing.T) {
	input := []byte("&amp;\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeHTMLInline {
		t.Fatalf("expected html_inline for entity, got %v", p.FirstChild)
	}
	if p.FirstChild.Data != "&amp;" {
		t.Fatalf("expected entity data '&amp;', got %q", p.FirstChild.Data)
	}
}

func TestHardBreak(t *testing.T) {
	input := []byte("hello  \nworld\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeParagraph {
		t.Fatal("expected paragraph")
	}
	p := root.FirstChild
	foundBreak := false
	for child := p.FirstChild; child != nil; child = child.Next {
		if child.Type == ast.NodeLinebreak {
			foundBreak = true
		}
	}
	if !foundBreak {
		t.Fatal("expected hard line break")
	}
}

func TestDebugHTMLBlock(t *testing.T) {
	p := New(0)
	p.Feed([]byte("<div>\nhello\n\n"))
	root := p.Finish()
	for child := root.FirstChild; child != nil; child = child.Next {
		t.Logf("Root child type=%d data=%q", child.Type, child.Data)
		for sub := child.FirstChild; sub != nil; sub = sub.Next {
			t.Logf("  sub type=%d data=%q", sub.Type, sub.Data)
		}
	}
}

func TestDebugOrderedListLineByLine(t *testing.T) {
	p := New(0)
	p.processLine([]byte("1. first\n"))
	t.Logf("After line 1: content=%q current type=%d", p.content.String(), p.current.Type)
	p.processLine([]byte("2. second\n"))
	t.Logf("After line 2: content=%q current type=%d", p.content.String(), p.current.Type)
}

func TestHTMLBlockType7Termination(t *testing.T) {
	input := []byte("<div>\nhello\n\n")
	root, _ := ParseDocument(input, 0)
	for child := root.FirstChild; child != nil; child = child.Next {
		t.Logf("Root child type=%d data=%q", child.Type, child.Data)
		for sub := child.FirstChild; sub != nil; sub = sub.Next {
			t.Logf("  sub type=%d data=%q", sub.Type, sub.Data)
		}
	}
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeHTMLBlock {
		t.Fatalf("expected html block, got %v", root.FirstChild)
	}
	html := root.FirstChild
	if html.FirstChild != nil {
		t.Fatal("expected html block to have no children")
	}
	if html.Data != "<div>\nhello\n" {
		t.Fatalf("expected html block data '<div>\\nhello\\n', got %q", html.Data)
	}
	if html.Next != nil {
		t.Fatalf("expected no siblings after html block, got %v", html.Next)
	}
}

func TestFencedCodeBlockInfoString(t *testing.T) {
	input := []byte("```go\ncode\n```\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeCodeBlock {
		t.Fatalf("expected code block, got %v", root.FirstChild)
	}
	cb := root.FirstChild
	if cb.CodeData == nil || cb.CodeData.Info != "go" {
		t.Fatalf("expected info 'go', got %q", cb.CodeData.Info)
	}
	if cb.Data != "code\n" {
		t.Fatalf("expected data 'code\\n', got %q", cb.Data)
	}
}

func TestSetextVsThematicBreak(t *testing.T) {
	// A line of dashes after a paragraph should be a setext heading, not a thematic break.
	input := []byte("hello\n---\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeHeading {
		t.Fatalf("expected heading, got %v", root.FirstChild)
	}
	h := root.FirstChild
	if h.HeadingData == nil || h.HeadingData.Level != 2 || !h.HeadingData.Setext {
		t.Fatalf("expected setext h2, got %v", h.HeadingData)
	}
	if h.FirstChild == nil || h.FirstChild.Type != ast.NodeText || h.FirstChild.Data != "hello" {
		t.Fatalf("expected heading text 'hello', got %v", h.FirstChild)
	}
}

func TestBlockQuoteLazyContinuation(t *testing.T) {
	input := []byte("> line1\nlazy\n")
	root, _ := ParseDocument(input, 0)
	if root.FirstChild == nil || root.FirstChild.Type != ast.NodeBlockQuote {
		t.Fatalf("expected block quote, got %v", root.FirstChild)
	}
	bq := root.FirstChild
	if bq.FirstChild == nil || bq.FirstChild.Type != ast.NodeParagraph {
		t.Fatalf("expected paragraph inside block quote, got %v", bq.FirstChild)
	}
	p := bq.FirstChild
	if p.FirstChild == nil || p.FirstChild.Type != ast.NodeText || p.FirstChild.Data != "line1" {
		t.Fatalf("expected first text 'line1', got %v", p.FirstChild)
	}
	if p.FirstChild.Next == nil || p.FirstChild.Next.Type != ast.NodeSoftbreak {
		t.Fatalf("expected softbreak after 'line1', got %v", p.FirstChild.Next)
	}
	if p.FirstChild.Next.Next == nil || p.FirstChild.Next.Next.Type != ast.NodeText || p.FirstChild.Next.Next.Data != "lazy" {
		t.Fatalf("expected second text 'lazy', got %v", p.FirstChild.Next.Next)
	}
}
