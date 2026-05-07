package parser

import "github.com/briansumma/cmark/pkg/ast"

type delimiter struct {
	prev      *delimiter
	next      *delimiter
	inl       *ast.Node
	char      byte
	numDelims int
	origNum   int
	active    bool
	canOpen   bool
	canClose  bool
	position  int
}

type bracket struct {
	prev         *bracket
	node         *ast.Node
	position     int
	image        bool
	active       bool
	bracketAfter bool
}

func (s *subject) pushDelimiter(c byte, canOpen, canClose bool, inl *ast.Node) {
	d := &delimiter{
		prev:      s.lastDelim,
		inl:       inl,
		char:      c,
		numDelims: len(inl.Data),
		origNum:   len(inl.Data),
		active:    true,
		canOpen:   canOpen,
		canClose:  canClose,
		position:  s.pos - len(inl.Data),
	}
	if d.prev != nil {
		d.prev.next = d
	}
	s.lastDelim = d
}

func (s *subject) removeDelimiter(d *delimiter) {
	if d == nil {
		return
	}
	if d.next == nil {
		s.lastDelim = d.prev
	} else {
		d.next.prev = d.prev
	}
	if d.prev != nil {
		d.prev.next = d.next
	}
}

func (s *subject) popBracket() {
	if s.lastBracket == nil {
		return
	}
	s.lastBracket = s.lastBracket.prev
}

func (s *subject) pushBracket(image bool, inl *ast.Node) {
	if s.lastBracket != nil {
		s.lastBracket.bracketAfter = true
	}
	b := &bracket{
		prev:     s.lastBracket,
		node:     inl,
		position: s.pos,
		image:    image,
		active:   true,
	}
	s.lastBracket = b
	if !image {
		s.noLinkOpeners = false
	}
}
