// Package utf8 provides locale-independent UTF-8 utilities.
package utf8

import (
	"strings"
	"unicode"

	"github.com/briansumma/cmark/internal/buffer"
)

var utf8Class = [256]int8{
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 4, 4, 4, 0, 0, 0, 0, 0, 0, 0, 0,
}

func encodeUnknown(buf *buffer.Buffer) {
	buf.Append([]byte{0xEF, 0xBF, 0xBD})
}

func utf8CharLen(str []byte) int {
	if len(str) == 0 {
		return 0
	}
	length := utf8Class[str[0]]
	if length == 0 {
		return -1
	}
	if int(length) > len(str) {
		return -len(str)
	}
	for i := 1; i < int(length); i++ {
		if (str[i] & 0xC0) != 0x80 {
			return -i
		}
	}
	return int(length)
}

func utf8Valid(str []byte) int {
	length := utf8Class[str[0]]
	if length == 0 {
		return -1
	}
	if int(length) > len(str) {
		return -len(str)
	}
	switch length {
	case 2:
		if (str[1] & 0xC0) != 0x80 {
			return -1
		}
		if str[0] < 0xC2 {
			return -2
		}
	case 3:
		if (str[1] & 0xC0) != 0x80 {
			return -1
		}
		if (str[2] & 0xC0) != 0x80 {
			return -2
		}
		if str[0] == 0xE0 {
			if str[1] < 0xA0 {
				return -3
			}
		} else if str[0] == 0xED {
			if str[1] >= 0xA0 {
				return -3
			}
		}
	case 4:
		if (str[1] & 0xC0) != 0x80 {
			return -1
		}
		if (str[2] & 0xC0) != 0x80 {
			return -2
		}
		if (str[3] & 0xC0) != 0x80 {
			return -3
		}
		if str[0] == 0xF0 {
			if str[1] < 0x90 {
				return -4
			}
		} else if str[0] >= 0xF4 {
			if str[0] > 0xF4 || str[1] >= 0x90 {
				return -4
			}
		}
	}
	return int(length)
}

// Check validates UTF-8, replacing invalid sequences with U+FFFD.
func Check(dest *buffer.Buffer, line []byte) {
	i := 0
	for i < len(line) {
		org := i
		charlen := 0
		for i < len(line) {
			if line[i] < 0x80 && line[i] != 0 {
				i++
			} else if line[i] >= 0x80 {
				charlen = utf8Valid(line[i:])
				if charlen < 0 {
					charlen = -charlen
					break
				}
				i += charlen
			} else if line[i] == 0 {
				charlen = 1
				break
			}
		}
		if i > org {
			dest.Append(line[org:i])
		}
		if i >= len(line) {
			break
		}
		encodeUnknown(dest)
		i += charlen
	}
}

// Iterate returns the next rune and its byte length from str.
func Iterate(str []byte) (rune, int) {
	length := utf8CharLen(str)
	if length < 0 {
		return 0xFFFD, 1
	}
	if length == 0 {
		return 0, 0
	}
	var uc rune
	switch length {
	case 1:
		uc = rune(str[0])
	case 2:
		uc = (rune(str[0]&0x1F) << 6) | rune(str[1]&0x3F)
		if uc < 0x80 {
			return 0xFFFD, length
		}
	case 3:
		uc = (rune(str[0]&0x0F) << 12) | (rune(str[1]&0x3F) << 6) | rune(str[2]&0x3F)
		if uc < 0x800 || (uc >= 0xD800 && uc < 0xE000) {
			return 0xFFFD, length
		}
	case 4:
		uc = (rune(str[0]&0x07) << 18) | (rune(str[1]&0x3F) << 12) |
			(rune(str[2]&0x3F) << 6) | rune(str[3]&0x3F)
		if uc < 0x10000 || uc >= 0x110000 {
			return 0xFFFD, length
		}
	}
	return uc, length
}

// EncodeChar writes the UTF-8 encoding of rune r into buf.
func EncodeChar(r rune, buf *buffer.Buffer) {
	if r < 0x80 {
		buf.AppendByte(byte(r))
	} else if r < 0x800 {
		buf.AppendByte(0xC0 + byte(r>>6))
		buf.AppendByte(0x80 + byte(r&0x3F))
	} else if r < 0x10000 {
		buf.AppendByte(0xE0 + byte(r>>12))
		buf.AppendByte(0x80 + byte((r>>6)&0x3F))
		buf.AppendByte(0x80 + byte(r&0x3F))
	} else if r < 0x110000 {
		buf.AppendByte(0xF0 + byte(r>>18))
		buf.AppendByte(0x80 + byte((r>>12)&0x3F))
		buf.AppendByte(0x80 + byte((r>>6)&0x3F))
		buf.AppendByte(0x80 + byte(r&0x3F))
	} else {
		encodeUnknown(buf)
	}
}

// CaseFold folds the UTF-8 string into dest.
func CaseFold(dest *buffer.Buffer, str []byte) {
	for len(str) > 0 {
		r, charlen := Iterate(str)
		if charlen == 1 {
			if r >= 'A' && r <= 'Z' {
				r += 'a' - 'A'
			}
			dest.AppendByte(byte(r))
		} else if charlen > 0 {
			s := strings.ToLower(string(str[:charlen]))
			dest.AppendString(s)
		} else {
			encodeUnknown(dest)
			charlen = 1
		}
		str = str[charlen:]
	}
}

// IsSpace reports whether r is a Unicode space character.
func IsSpace(r rune) bool {
	return r == 9 || r == 10 || r == 12 || r == 13 || r == 32 ||
		r == 160 || r == 5760 || (r >= 8192 && r <= 8202) ||
		r == 8239 || r == 8287 || r == 12288
}

// IsPunctuationOrSymbol reports whether r is punctuation or a symbol.
func IsPunctuationOrSymbol(r rune) bool {
	return unicode.IsPunct(rune(r)) || unicode.Is(unicode.S, rune(r))
}
