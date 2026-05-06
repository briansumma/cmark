// Package utf8 provides locale-independent UTF-8 utilities.
package utf8

import "github.com/briansumma/cmark/internal/buffer"

// CaseFold folds the UTF-8 string into dest.
func CaseFold(dest *buffer.Buffer, str []byte) {
	// TODO: stub
}

// EncodeChar writes the UTF-8 encoding of rune r into buf.
func EncodeChar(r rune, buf *buffer.Buffer) {
	// TODO: stub
}

// Iterate returns the next rune and its byte length from str.
// If the string is empty or invalid, it returns RuneError and 0 or 1.
func Iterate(str []byte) (rune, int) {
	// TODO: stub
	return 0, 0
}

// Check validates UTF-8, replacing invalid sequences with U+FFFD.
func Check(dest *buffer.Buffer, line []byte) {
	// TODO: stub
}

// IsSpace reports whether r is a Unicode space character.
func IsSpace(r rune) bool {
	// TODO: stub
	return false
}

// IsPunctuationOrSymbol reports whether r is punctuation or a symbol.
func IsPunctuationOrSymbol(r rune) bool {
	// TODO: stub
	return false
}
