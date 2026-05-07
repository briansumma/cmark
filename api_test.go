package cmark

import "testing"

func TestMarkdownToHTML(t *testing.T) {
	got := MarkdownToHTML("# Hello\n", 0)
	want := "<h1>Hello</h1>\n"
	if got != want {
		t.Errorf("MarkdownToHTML = %q, want %q", got, want)
	}
}
