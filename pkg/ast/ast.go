// Package ast defines the CommonMark abstract syntax tree types and operations.
package ast

import "fmt"

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

// NodeTypeString returns a human-readable name for the node type.
func NodeTypeString(t NodeType) string {
	switch t {
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
func (n *Node) Dump(w fmt.Stringer, level int) {
	// TODO: implement tree dump for debugging.
}

// NewNode creates a new node of the given type.
func NewNode(t NodeType) *Node {
	return &Node{Type: t}
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

// InsertBefore inserts sibling before node.
func (n *Node) InsertBefore(sibling *Node) bool {
	// TODO: stub
	return false
}

// InsertAfter inserts sibling after node.
func (n *Node) InsertAfter(sibling *Node) bool {
	// TODO: stub
	return false
}

// Replace replaces oldnode with newnode and unlinks oldnode.
func Replace(oldnode, newnode *Node) bool {
	// TODO: stub
	return false
}

// PrependChild adds child to the beginning of the children of node.
func (n *Node) PrependChild(child *Node) bool {
	// TODO: stub
	return false
}

// AppendChild adds child to the end of the children of node.
func (n *Node) AppendChild(child *Node) bool {
	// TODO: stub
	return false
}

// ConsolidateTextNodes merges adjacent text nodes.
func ConsolidateTextNodes(root *Node) {
	// TODO: stub
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
	return &Iter{root: root}
}

// Next advances to the next node and returns the event type.
func (it *Iter) Next() EventType {
	// TODO: stub
	return EventDone
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

// Reset resets the iterator to the given node and event type.
func (it *Iter) Reset(current *Node, evType EventType) {
	it.cur = iterState{node: current, evType: evType}
}

// Reference is a single link reference entry.
type Reference struct {
	Label string
	URL   string
	Title string
}

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

// Lookup finds a reference by label.
func (rm *ReferenceMap) Lookup(label string) *Reference {
	return rm.refs[label]
}

// Create adds a reference to the map.
func (rm *ReferenceMap) Create(label, url, title string) {
	rm.refs[label] = &Reference{Label: label, URL: url, Title: title}
}
