// Command cmark converts CommonMark Markdown to various output formats.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/briansumma/cmark"
	"github.com/briansumma/cmark/pkg/ast"
)

func main() {
	var (
		toFormat = flag.String("t", "html", "output format: html, xml, man, commonmark, latex")
		output   = flag.String("o", "", "write output to FILE instead of stdout")
		width    = flag.Int("width", 0, "wrap width for commonmark and man output")
		safe     = flag.Bool("safe", false, "strip raw HTML and dangerous URLs")
		help     = flag.Bool("help", false, "print usage information")
		version  = flag.Bool("version", false, "print version")
	)
	flag.StringVar(toFormat, "to", "html", "output format")
	flag.StringVar(output, "output", "", "write output to FILE")
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] [FILE...]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Convert CommonMark Markdown to HTML, XML, man, CommonMark, or LaTeX.")
		fmt.Fprintln(os.Stderr)
		flag.PrintDefaults()
		os.Exit(0)
	}
	if *version {
		fmt.Println(cmark.VersionString())
		os.Exit(0)
	}

	options := ast.OptDefault
	if *safe {
		options |= ast.OptSafe
	}

	var data []byte
	var err error
	if flag.NArg() == 0 {
		data, err = os.ReadFile("/dev/stdin")
	} else {
		data, err = os.ReadFile(flag.Arg(0))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	root, err := cmark.ParseDocument(data, options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing markdown: %v\n", err)
		os.Exit(1)
	}

	var out string
	switch *toFormat {
	case "html":
		out = cmark.RenderHTML(root, options)
	case "xml":
		out = cmark.RenderXML(root, options)
	case "man":
		out = cmark.RenderMan(root, options, *width)
	case "commonmark":
		out = cmark.RenderCommonmark(root, options, *width)
	case "latex":
		out = cmark.RenderLatex(root, options, *width)
	default:
		fmt.Fprintf(os.Stderr, "Unknown format: %s\n", *toFormat)
		os.Exit(1)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(out), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Print(out)
	}
}
