package main

import (
	"fmt"
	"github.com/briansumma/cmark/internal/parser"
	"github.com/briansumma/cmark/pkg/ast"
)

func printTree(n *ast.Node, indent int) {
	for i := 0; i < indent; i++ {
		fmt.Print("  ")
	}
	fmt.Printf("%s (lines %d-%d, data=%q)\n", n.Type, n.StartLine, n.EndLine, n.Data)
	for c := n.FirstChild; c != nil; c = c.Next {
		printTree(c, indent+1)
	}
}

func main() {
	data := []byte("* item 1\n* item 2\n")
	root, err := parser.ParseDocument(data, 0)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	printTree(root, 0)
}
