// Package ctype provides locale-independent character classification.
package ctype

// IsSpace reports whether c is a space character (space, tab, newline, etc.).
func IsSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' || c == '\v'
}

// IsPunct reports whether c is ASCII punctuation.
func IsPunct(c byte) bool {
	return (c >= '!' && c <= '/') ||
		(c >= ':' && c <= '@') ||
		(c >= '[' && c <= '`') ||
		(c >= '{' && c <= '~')
}

// IsDigit reports whether c is an ASCII digit.
func IsDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// IsAlpha reports whether c is an ASCII letter.
func IsAlpha(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}
