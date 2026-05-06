// Package ast defines the CommonMark abstract syntax tree types and operations.
package ast

import (
	"fmt"
	"io"
	"strings"
)

// NodeType represents the type of a node in the AST.
type NodeType int

// Node type constants.
const (
	NodeNone NodeType = iota
	NodeDocument
	NodeBlockQuote
	NodeList
	NodeItem
	NodeCodeBlock
	NodeHTMLBlock
	NodeCustomBlock
	NodeParagraph
	NodeHeading
	NodeThematicBreak
	NodeText
	NodeSoftbreak
	NodeLinebreak
	NodeCode
	NodeHTMLInline
	NodeCustomInline
	NodeEmph
	NodeStrong
	NodeLink
	NodeImage
)

// Sentinel values for block / inline ranges (mirroring C constants).
const (
	NodeFirstBlock  = NodeDocument
	NodeLastBlock   = NodeThematicBreak
	NodeFirstInline = NodeText
	NodeLastInline  = NodeImage
)

// Internal flags.
const (
	NodeOpen              = 1 << 0
	NodeLastLineBlank     = 1 << 1
	NodeLastLineChecked   = 1 << 2
	NodeListLastLineBlank = 1 << 3
)

// NodeTypeString returns a human-readable name for the node type.
func NodeTypeString(t NodeType) string {
	switch t {
	case NodeNone:
		return "none"
	case NodeDocument:
		return "document"
	case NodeBlockQuote:
		return "block_quote"
	case NodeList:
		return "list"
	case NodeItem:
		return "item"
	case NodeCodeBlock:
		return "code_block"
	case NodeHTMLBlock:
		return "html_block"
	case NodeCustomBlock:
		return "custom_block"
	case NodeParagraph:
		return "paragraph"
	case NodeHeading:
		return "heading"
	case NodeThematicBreak:
		return "thematic_break"
	case NodeText:
		return "text"
	case NodeSoftbreak:
		return "softbreak"
	case NodeLinebreak:
		return "linebreak"
	case NodeCode:
		return "code"
	case NodeHTMLInline:
		return "html_inline"
	case NodeCustomInline:
		return "custom_inline"
	case NodeEmph:
		return "emph"
	case NodeStrong:
		return "strong"
	case NodeLink:
		return "link"
	case NodeImage:
		return "image"
	default:
		return "<unknown>"
	}
}

// ListType specifies the kind of list.
type ListType int

const (
	NoList ListType = iota
	BulletList
	OrderedList
)

// DelimType specifies the list delimiter.
type DelimType int

const (
	NoDelim DelimType = iota
	PeriodDelim
	ParenDelim
)

// EventType is returned by the tree iterator.
type EventType int

const (
	EventNone EventType = iota
	EventDone
	EventEnter
	EventExit
)

// ListData holds list-specific node data.
type ListData struct {
	MarkerOffset int
	Padding      int
	Start        int
	ListType     ListType
	Delimiter    DelimType
	BulletChar   byte
	Tight        bool
}

// CodeData holds fenced-code-block-specific node data.
type CodeData struct {
	Info        string
	FenceLength uint8
	FenceOffset uint8
	FenceChar   byte
	Fenced      bool
}

// HeadingData holds heading-specific node data.
type HeadingData struct {
	InternalOffset int
	Level          int8
	Setext         bool
}

// LinkData holds link/image-specific node data.
type LinkData struct {
	URL   string
	Title string
}

// CustomData holds custom node literal text.
type CustomData struct {
	OnEnter string
	OnExit  string
}

// Node is a single node in the CommonMark AST.
type Node struct {
	Type        NodeType
	Flags       uint16
	Data        string
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	UserData    any

	Next       *Node
	Prev       *Node
	Parent     *Node
	FirstChild *Node
	LastChild  *Node

	ListData      *ListData
	CodeData      *CodeData
	HeadingData   *HeadingData
	LinkData      *LinkData
	CustomData    *CustomData
	HTMLBlockType int
}

// IsBlock reports whether the node is a block node.
func (n *Node) IsBlock() bool {
	return n.Type >= NodeDocument && n.Type <= NodeThematicBreak
}

// IsInline reports whether the node is an inline node.
func (n *Node) IsInline() bool {
	return n.Type >= NodeText && n.Type <= NodeImage
}

// IsLeaf reports whether the node is a leaf node.
func (n *Node) IsLeaf() bool {
	switch n.Type {
	case NodeHTMLBlock, NodeThematicBreak, NodeCodeBlock,
		NodeText, NodeSoftbreak, NodeLinebreak, NodeCode, NodeHTMLInline:
		return true
	}
	return false
}

// String returns a short textual representation for debugging.
func (n *Node) String() string {
	if n == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s [%d:%d-%d:%d]", NodeTypeString(n.Type),
		n.StartLine, n.StartColumn, n.EndLine, n.EndColumn)
}

// Dump writes a detailed representation of the subtree to a file-like writer.
func (n *Node) Dump(w io.Writer, level int) {
	if n == nil {
		return
	}
	indent := strings.Repeat("  ", level)
	fmt.Fprintf(w, "%s%s", indent, NodeTypeString(n.Type))
	switch n.Type {
	case NodeText, NodeCode, NodeCodeBlock, NodeHTMLBlock, NodeHTMLInline:
		if n.Data != "" {
			fmt.Fprintf(w, " %q", n.Data)
		}
	case NodeHeading:
		if n.HeadingData != nil {
			fmt.Fprintf(w, " level=%d", n.HeadingData.Level)
		}
	case NodeList:
		if n.ListData != nil {
			fmt.Fprintf(w, " type=%s tight=%v",
				listTypeString(n.ListData.ListType), n.ListData.Tight)
		}
	}
	fmt.Fprintf(w, " [%d:%d-%d:%d]\n",
		n.StartLine, n.StartColumn, n.EndLine, n.EndColumn)
	for child := n.FirstChild; child != nil; child = child.Next {
		child.Dump(w, level+1)
	}
}

func listTypeString(t ListType) string {
	switch t {
	case BulletList:
		return "bullet"
	case OrderedList:
		return "ordered"
	default:
		return "none"
	}
}

// NewNode creates a new node of the given type.
func NewNode(t NodeType) *Node {
	n := &Node{Type: t}
	switch t {
	case NodeHeading:
		n.HeadingData = &HeadingData{Level: 1}
	case NodeList:
		n.ListData = &ListData{
			ListType:   BulletList,
			Start:      0,
			Tight:      false,
			BulletChar: '*',
		}
	}
	return n
}

// Free recursively frees a node and its children.
func Free(n *Node) {
	// In Go we rely on GC; this is a no-op for API compatibility.
}

// Unlink removes a node from the tree without freeing it.
func (n *Node) Unlink() {
	if n.Prev != nil {
		n.Prev.Next = n.Next
	} else if n.Parent != nil {
		n.Parent.FirstChild = n.Next
	}
	if n.Next != nil {
		n.Next.Prev = n.Prev
	} else if n.Parent != nil {
		n.Parent.LastChild = n.Prev
	}
	n.Parent = nil
	n.Prev = nil
	n.Next = nil
}

// canContain reports whether child can be inserted into node.
func canContain(node, child *Node) bool {
	if node == nil || child == nil || node == child {
		return false
	}
	// Verify that child is not an ancestor of node.
	if child.FirstChild != nil {
		for cur := node.Parent; cur != nil; cur = cur.Parent {
			if cur == child {
				return false
			}
		}
	}
	if child.Type == NodeDocument {
		return false
	}
	switch node.Type {
	case NodeDocument, NodeBlockQuote, NodeItem:
		return child.IsBlock() && child.Type != NodeItem
	case NodeList:
		return child.Type == NodeItem
	case NodeCustomBlock:
		return true
	case NodeParagraph, NodeHeading, NodeEmph, NodeStrong, NodeLink,
		NodeImage, NodeCustomInline:
		return child.IsInline()
	default:
		return false
	}
}

// InsertBefore inserts sibling before node.
func (n *Node) InsertBefore(sibling *Node) bool {
	if n == nil || sibling == nil || n.Parent == nil {
		return false
	}
	if !canContain(n.Parent, sibling) {
		return false
	}
	sibling.Unlink()
	oldPrev := n.Prev
	if oldPrev != nil {
		oldPrev.Next = sibling
	}
	sibling.Prev = oldPrev
	sibling.Next = n
	n.Prev = sibling
	parent := n.Parent
	sibling.Parent = parent
	if oldPrev == nil && parent != nil {
		parent.FirstChild = sibling
	}
	return true
}

// InsertAfter inserts sibling after node.
func (n *Node) InsertAfter(sibling *Node) bool {
	if n == nil || sibling == nil || n.Parent == nil {
		return false
	}
	if !canContain(n.Parent, sibling) {
		return false
	}
	sibling.Unlink()
	oldNext := n.Next
	if oldNext != nil {
		oldNext.Prev = sibling
	}
	sibling.Next = oldNext
	sibling.Prev = n
	n.Next = sibling
	parent := n.Parent
	sibling.Parent = parent
	if oldNext == nil && parent != nil {
		parent.LastChild = sibling
	}
	return true
}

// Replace replaces oldnode with newnode and unlinks oldnode.
func Replace(oldnode, newnode *Node) bool {
	if !oldnode.InsertBefore(newnode) {
		return false
	}
	oldnode.Unlink()
	return true
}

// PrependChild adds child to the beginning of the children of node.
func (n *Node) PrependChild(child *Node) bool {
	if !canContain(n, child) {
		return false
	}
	child.Unlink()
	oldFirst := n.FirstChild
	child.Next = oldFirst
	child.Prev = nil
	child.Parent = n
	n.FirstChild = child
	if oldFirst != nil {
		oldFirst.Prev = child
	} else {
		n.LastChild = child
	}
	return true
}

// AppendChild adds child to the end of the children of node.
func (n *Node) AppendChild(child *Node) bool {
	if !canContain(n, child) {
		return false
	}
	child.Unlink()
	oldLast := n.LastChild
	child.Next = nil
	child.Prev = oldLast
	child.Parent = n
	n.LastChild = child
	if oldLast != nil {
		oldLast.Next = child
	} else {
		n.FirstChild = child
	}
	return true
}

// Iter walks through a tree of nodes.
type Iter struct {
	root *Node
	cur  iterState
	next iterState
}

type iterState struct {
	evType EventType
	node   *Node
}

// NewIter creates a new iterator starting at root.
func NewIter(root *Node) *Iter {
	if root == nil {
		return nil
	}
	return &Iter{
		root: root,
		next: iterState{evType: EventEnter, node: root},
	}
}

// Next advances to the next node and returns the event type.
func (it *Iter) Next() EventType {
	evType := it.next.evType
	node := it.next.node

	it.cur = it.next

	if evType == EventDone {
		return evType
	}

	if evType == EventEnter && !node.IsLeaf() {
		if node.FirstChild == nil {
			it.next = iterState{evType: EventExit, node: node}
		} else {
			it.next = iterState{evType: EventEnter, node: node.FirstChild}
		}
	} else if node == it.root {
		it.next = iterState{evType: EventDone, node: nil}
	} else if node.Next != nil {
		it.next = iterState{evType: EventEnter, node: node.Next}
	} else if node.Parent != nil {
		it.next = iterState{evType: EventExit, node: node.Parent}
	} else {
		it.next = iterState{evType: EventDone, node: nil}
	}

	return evType
}

// GetNode returns the current node.
func (it *Iter) GetNode() *Node {
	return it.cur.node
}

// GetEventType returns the current event type.
func (it *Iter) GetEventType() EventType {
	return it.cur.evType
}

// GetRoot returns the root node.
func (it *Iter) GetRoot() *Node {
	return it.root
}

// Reset resets the iterator so that the current node is current and
// the event type is evType.
func (it *Iter) Reset(current *Node, evType EventType) {
	it.next = iterState{node: current, evType: evType}
	it.Next()
}

// ConsolidateTextNodes merges adjacent text nodes.
func ConsolidateTextNodes(root *Node) {
	if root == nil {
		return
	}
	it := NewIter(root)
	for evType := it.Next(); evType != EventDone; evType = it.Next() {
		cur := it.GetNode()
		if evType == EventEnter && cur.Type == NodeText &&
			cur.Next != nil && cur.Next.Type == NodeText {
			var b strings.Builder
			b.WriteString(cur.Data)
			tmp := cur.Next
			for tmp != nil && tmp.Type == NodeText {
				it.Next() // advance past tmp
				b.WriteString(tmp.Data)
				cur.EndColumn = tmp.EndColumn
				next := tmp.Next
				tmp.Unlink()
				tmp = next
			}
			cur.Data = b.String()
		}
	}
}

// Reference is a single link reference entry.
type Reference struct {
	Label string
	URL   string
	Title string
}

// MaxLinkLabelLength is the maximum length of a link label.
const MaxLinkLabelLength = 1000

// Options affect parsing and rendering behavior.
type Options int

const (
	OptDefault      Options = 0
	OptSourcePos    Options = 1 << 1
	OptHardBreaks   Options = 1 << 2
	OptSafe         Options = 1 << 3
	OptNoBreaks     Options = 1 << 4
	OptNormalize    Options = 1 << 8
	OptValidateUTF8 Options = 1 << 9
	OptSmart        Options = 1 << 10
	OptUnsafe       Options = 1 << 17
)

// ReferenceMap stores and looks up link references.
type ReferenceMap struct {
	refs map[string]*Reference
}

// NewReferenceMap creates a new reference map.
func NewReferenceMap() *ReferenceMap {
	return &ReferenceMap{refs: make(map[string]*Reference)}
}

// isSpace reports whether c is an ASCII whitespace character.
func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// normalizeReference normalizes a reference label by case folding, trimming,
// and collapsing internal whitespace.  It returns the empty string if the
// label is composed solely of whitespace.
func normalizeReference(label string) string {
	// Case fold.
	label = strings.ToLower(label)
	// Trim leading/trailing whitespace.
	start := 0
	for start < len(label) && isSpace(label[start]) {
		start++
	}
	end := len(label)
	for end > start && isSpace(label[end-1]) {
		end--
	}
	label = label[start:end]
	// Collapse consecutive whitespace to a single space.
	var b strings.Builder
	b.Grow(len(label))
	lastWasSpace := false
	for i := 0; i < len(label); i++ {
		c := label[i]
		if isSpace(c) {
			if !lastWasSpace && b.Len() > 0 {
				b.WriteByte(' ')
			}
			lastWasSpace = true
		} else {
			b.WriteByte(c)
			lastWasSpace = false
		}
	}
	result := b.String()
	if result == "" {
		return ""
	}
	return result
}

// Lookup finds a reference by label.
func (rm *ReferenceMap) Lookup(label string) *Reference {
	if len(label) < 1 || len(label) > MaxLinkLabelLength {
		return nil
	}
	if rm == nil || len(rm.refs) == 0 {
		return nil
	}
	norm := normalizeReference(label)
	if norm == "" {
		return nil
	}
	return rm.refs[norm]
}

// Create adds a reference to the map.
func (rm *ReferenceMap) Create(label, url, title string) {
	if rm == nil {
		return
	}
	norm := normalizeReference(label)
	if norm == "" {
		return
	}
	if _, exists := rm.refs[norm]; !exists {
		rm.refs[norm] = &Reference{Label: norm, URL: url, Title: title}
	}
}
