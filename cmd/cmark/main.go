// Command cmark converts CommonMark Markdown to HTML.
package main

import (
	"fmt"
	"os"

	"github.com/briansumma/cmark"
)

func main() {
	data, err := os.ReadFile("/dev/stdin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
		os.Exit(1)
	}

	html, err := cmark.MarkdownToHTML(data, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing markdown:", err)
		os.Exit(1)
	}

	fmt.Print(html)
}
