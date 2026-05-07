package parser

import (
	"fmt"
	"testing"
)

func TestParse(t *testing.T) {
	data := []byte("# Hello\n\nworld.\n")
	fmt.Println("Parsing...")
	root, err := ParseDocument(data, 0)
	fmt.Println("Done parsing, err:", err)
	if root != nil {
		fmt.Println("Root type:", root.Type)
		for child := root.FirstChild; child != nil; child = child.Next {
			fmt.Println("Child type:", child.Type, "data:", child.Data)
		}
	}
}
