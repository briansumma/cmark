// Package scanners provides hand-rolled byte-level pattern scanners.
// These replace the re2c-generated C scanners for the CommonMark parser.
package scanners

import (
	"bytes"
	"strings"
)

// spacechar: [ \t\v\f\r\n]
func isSpacechar(c byte) bool {
	return c == ' ' || c == '\t' || c == '\v' || c == '\f' || c == '\r' || c == '\n'
}

// ScanScheme scans an autolink scheme at the start of p.
// Returns the number of bytes consumed (including the colon), or 0 if no match.
func ScanScheme(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if !(c >= 'A' && c <= 'Z') && !(c >= 'a' && c <= 'z') {
		return 0
	}
	i := 1
	for i < len(p) {
		c = p[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '+' || c == '-' {
			i++
			continue
		}
		break
	}
	if i < 2 || i > 32 {
		return 0
	}
	if i >= len(p) || p[i] != ':' {
		return 0
	}
	return i + 1
}

// ScanAutolinkURI scans an autolink URI: scheme:[^\x00-\x20<>]*>
func ScanAutolinkURI(p []byte) int {
	n := ScanScheme(p)
	if n == 0 {
		return 0
	}
	i := n
	for i < len(p) {
		c := p[i]
		if c == 0 || c <= ' ' || c == '<' || c == '>' {
			break
		}
		i++
	}
	if i >= len(p) || p[i] != '>' {
		return 0
	}
	return i + 1
}

// ScanAutolinkEmail scans an autolink email.
func ScanAutolinkEmail(p []byte) int {
	i := 0
	for i < len(p) {
		c := p[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '.' || c == '!' || c == '#' ||
			c == '$' || c == '%' || c == '&' || c == '\'' || c == '*' ||
			c == '+' || c == '/' || c == '=' || c == '?' || c == '^' ||
			c == '_' || c == '`' || c == '{' || c == '|' || c == '}' ||
			c == '~' || c == '-' {
			i++
		} else {
			break
		}
	}
	if i == 0 || i >= len(p) || p[i] != '@' {
		return 0
	}
	i++ // consume '@'

	// domain label: [a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?
	if i >= len(p) {
		return 0
	}
	c := p[i]
	if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
		return 0
	}
	i++
	start := i
	for i < len(p) {
		c = p[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' {
			i++
		} else {
			break
		}
	}
	if i-start > 61 {
		return 0
	}
	if i > start {
		c = p[i-1]
		if c == '-' {
			return 0
		}
	}

	// more labels: ([.][a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*
	for i < len(p) && p[i] == '.' {
		j := i + 1
		if j >= len(p) {
			return 0
		}
		c = p[j]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
			return 0
		}
		j++
		start2 := j
		for j < len(p) {
			c = p[j]
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
				(c >= '0' && c <= '9') || c == '-' {
				j++
			} else {
				break
			}
		}
		if j-start2 > 61 {
			return 0
		}
		if j > start2 {
			if p[j-1] == '-' {
				return 0
			}
		}
		i = j
	}

	if i >= len(p) || p[i] != '>' {
		return 0
	}
	return i + 1
}

// scanTagname matches [A-Za-z][A-Za-z0-9-]* and returns the length.
func scanTagname(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
		return 0
	}
	i := 1
	for i < len(p) {
		c = p[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == '-' {
			i++
		} else {
			break
		}
	}
	return i
}

// block tag names from the re2c spec.
var blockTagNames = []string{
	"address", "article", "aside", "base", "basefont", "blockquote", "body",
	"caption", "center", "col", "colgroup", "dd", "details", "dialog", "dir",
	"div", "dl", "dt", "fieldset", "figcaption", "figure", "footer", "form",
	"frame", "frameset", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header",
	"hr", "html", "iframe", "legend", "li", "link", "main", "menu", "menuitem",
	"nav", "noframes", "ol", "optgroup", "option", "p", "param", "section",
	"search", "title", "summary", "table", "tbody", "td", "tfoot", "th",
	"thead", "title", "tr", "track", "ul",
}

func isBlockTag(name []byte) bool {
	s := string(name)
	for _, tag := range blockTagNames {
		if tag == s {
			return true
		}
	}
	return false
}

// scanAttributeName matches [a-zA-Z_:][a-zA-Z0-9:._-]*
func scanAttributeName(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || c == '_' || c == ':') {
		return 0
	}
	i := 1
	for i < len(p) {
		c = p[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
			(c >= '0' && c <= '9') || c == ':' || c == '.' || c == '_' || c == '-' {
			i++
		} else {
			break
		}
	}
	return i
}

// scanAttributeValueSpec matches spacechar* = spacechar* attributevalue.
func scanAttributeValueSpec(p []byte) int {
	i := 0
	for i < len(p) && isSpacechar(p[i]) {
		i++
	}
	if i >= len(p) || p[i] != '=' {
		return 0
	}
	i++
	for i < len(p) && isSpacechar(p[i]) {
		i++
	}
	n := scanAttributeValue(p[i:])
	if n == 0 {
		return 0
	}
	return i + n
}

// scanAttributeValue matches unquoted, single-quoted, or double-quoted value.
func scanAttributeValue(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	// singlequotedvalue = ['][^'\x00]*[']
	if p[0] == '\'' {
		i := 1
		for i < len(p) && p[i] != '\'' && p[i] != 0 {
			i++
		}
		if i >= len(p) || p[i] != '\'' {
			return 0
		}
		return i + 1
	}
	// doublequotedvalue = ["][^"\x00]*["]
	if p[0] == '"' {
		i := 1
		for i < len(p) && p[i] != '"' && p[i] != 0 {
			i++
		}
		if i >= len(p) || p[i] != '"' {
			return 0
		}
		return i + 1
	}
	// unquotedvalue = [^ \t\r\n\v\f"'=<>`\x00]+
	i := 0
	for i < len(p) {
		c := p[i]
		if c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\v' ||
			c == '\f' || c == '"' || c == '\'' || c == '=' || c == '<' ||
			c == '>' || c == '`' || c == 0 {
			break
		}
		i++
	}
	if i == 0 {
		return 0
	}
	return i
}

// scanAttribute matches spacechar+ attributename attributevaluespec?
func scanAttribute(p []byte) int {
	if len(p) == 0 || !isSpacechar(p[0]) {
		return 0
	}
	i := 1
	for i < len(p) && isSpacechar(p[i]) {
		i++
	}
	n := scanAttributeName(p[i:])
	if n == 0 {
		return 0
	}
	i += n
	m := scanAttributeValueSpec(p[i:])
	if m > 0 {
		i += m
	}
	return i
}

// scanOpentag matches tagname attribute* spacechar* /? >
func scanOpentag(p []byte) int {
	n := scanTagname(p)
	if n == 0 {
		return 0
	}
	i := n
	for {
		m := scanAttribute(p[i:])
		if m == 0 {
			break
		}
		i += m
	}
	for i < len(p) && isSpacechar(p[i]) {
		i++
	}
	if i < len(p) && p[i] == '/' {
		i++
	}
	if i >= len(p) || p[i] != '>' {
		return 0
	}
	return i + 1
}

// scanClosetag matches / tagname spacechar* >
func scanClosetag(p []byte) int {
	if len(p) == 0 || p[0] != '/' {
		return 0
	}
	n := scanTagname(p[1:])
	if n == 0 {
		return 0
	}
	i := 1 + n
	for i < len(p) && isSpacechar(p[i]) {
		i++
	}
	if i >= len(p) || p[i] != '>' {
		return 0
	}
	return i + 1
}

// ScanHTMLTag scans an HTML tag.
func ScanHTMLTag(p []byte) int {
	if n := scanOpentag(p); n > 0 {
		return n
	}
	return scanClosetag(p)
}

// ScanHTMLComment scans an HTML comment.
func ScanHTMLComment(p []byte) int {
	if len(p) < 4 || p[0] != '-' || p[1] != '-' {
		return 0
	}
	i := 2
	for i < len(p) {
		if i+2 < len(p) && p[i] == '-' && p[i+1] == '-' && p[i+2] == '>' {
			return i + 3
		}
		if p[i] == 0 {
			return 0
		}
		i++
	}
	return 0
}

// ScanHTMLPI scans an HTML processing instruction.
func ScanHTMLPI(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	i := 0
	for i < len(p) {
		if p[i] == '?' && i+1 < len(p) && p[i+1] == '>' {
			return i + 2
		}
		if p[i] == '>' {
			return i + 1
		}
		if p[i] == 0 {
			return 0
		}
		i++
	}
	return 0
}

// ScanHTMLDeclaration scans an HTML declaration.
func ScanHTMLDeclaration(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')) {
		return 0
	}
	i := 1
	for i < len(p) {
		c = p[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			i++
		} else {
			break
		}
	}
	if i == 1 {
		return 0
	}
	for i < len(p) {
		if p[i] == '>' {
			return i + 1
		}
		if p[i] == 0 {
			return 0
		}
		i++
	}
	return 0
}

// ScanHTMLCDATA scans a CDATA section.
func ScanHTMLCDATA(p []byte) int {
	prefix := []byte("CDATA[")
	if !bytes.HasPrefix(p, prefix) {
		return 0
	}
	i := len(prefix)
	for i < len(p) {
		if i+2 < len(p) && p[i] == ']' && p[i+1] == ']' && p[i+2] == '>' {
			return i + 3
		}
		if p[i] == 0 {
			return 0
		}
		i++
	}
	return 0
}

// ScanHTMLBlockStart scans for the start of an HTML block (types 1-6).
func ScanHTMLBlockStart(p []byte) int {
	if len(p) == 0 || p[0] != '<' {
		return 0
	}
	q := p[1:]

	// Type 1: <(script|pre|textarea|style)(spacechar|>)
	for _, tag := range []string{"script", "pre", "textarea", "style"} {
		if strings.HasPrefix(string(q), tag) {
			n := len(tag)
			if n < len(q) && (isSpacechar(q[n]) || q[n] == '>') {
				return 1
			}
		}
	}

	// Type 2: <!--
	if strings.HasPrefix(string(q), "!--") {
		return 2
	}

	// Type 3: <?
	if len(q) > 0 && q[0] == '?' {
		return 3
	}

	// Type 4: <![A-Za-z]
	if len(q) > 1 && q[0] == '!' && ((q[1] >= 'A' && q[1] <= 'Z') || (q[1] >= 'a' && q[1] <= 'z')) {
		return 4
	}

	// Type 5: <![CDATA[
	if strings.HasPrefix(string(q), "![CDATA[") {
		return 5
	}

	// Type 6: <[/]?blocktagname(spacechar|/?>)
	i := 0
	if len(q) > 0 && q[0] == '/' {
		i = 1
	}
	n := scanTagname(q[i:])
	if n > 0 && isBlockTag(q[i:i+n]) {
		m := i + n
		if m < len(q) && (isSpacechar(q[m]) || q[m] == '>' ||
			(m+1 < len(q) && q[m] == '/' && q[m+1] == '>')) {
			return 6
		}
	}

	return 0
}

// ScanHTMLBlockStart7 scans for the start of HTML block type 7.
func ScanHTMLBlockStart7(p []byte) int {
	if len(p) == 0 || p[0] != '<' {
		return 0
	}
	// ScanHTMLTag expects input without the leading <
	n := ScanHTMLTag(p[1:])
	if n == 0 {
		return 0
	}
	i := 1 + n // +1 for the skipped <
	for i < len(p) && (p[i] == ' ' || p[i] == '\t') {
		i++
	}
	if i < len(p) && (p[i] == '\r' || p[i] == '\n') {
		return 7
	}
	return 0
}

// ScanHTMLBlockEnd1 scans for the end of HTML block type 1.
func ScanHTMLBlockEnd1(p []byte) int {
	for i := 0; i < len(p); i++ {
		if p[i] == '<' && i+1 < len(p) && p[i+1] == '/' {
			rest := p[i+2:]
			for _, tag := range []string{"script", "pre", "textarea", "style"} {
				if strings.HasPrefix(string(rest), tag) {
					n := len(tag)
					if n < len(rest) && rest[n] == '>' {
						return i + 2 + n + 1
					}
				}
			}
		}
		if p[i] == 0 || p[i] == '\n' {
			return 0
		}
	}
	return 0
}

// ScanHTMLBlockEnd2 scans for the end of HTML block type 2.
func ScanHTMLBlockEnd2(p []byte) int {
	for i := 0; i < len(p); i++ {
		if i+2 < len(p) && p[i] == '-' && p[i+1] == '-' && p[i+2] == '>' {
			return i + 3
		}
		if p[i] == 0 || p[i] == '\n' {
			return 0
		}
	}
	return 0
}

// ScanHTMLBlockEnd3 scans for the end of HTML block type 3.
func ScanHTMLBlockEnd3(p []byte) int {
	for i := 0; i < len(p); i++ {
		if i+1 < len(p) && p[i] == '?' && p[i+1] == '>' {
			return i + 2
		}
		if p[i] == 0 || p[i] == '\n' {
			return 0
		}
	}
	return 0
}

// ScanHTMLBlockEnd4 scans for the end of HTML block type 4.
func ScanHTMLBlockEnd4(p []byte) int {
	for i := 0; i < len(p); i++ {
		if p[i] == '>' {
			return i + 1
		}
		if p[i] == 0 || p[i] == '\n' {
			return 0
		}
	}
	return 0
}

// ScanHTMLBlockEnd5 scans for the end of HTML block type 5.
func ScanHTMLBlockEnd5(p []byte) int {
	for i := 0; i < len(p); i++ {
		if i+2 < len(p) && p[i] == ']' && p[i+1] == ']' && p[i+2] == '>' {
			return i + 3
		}
		if p[i] == 0 || p[i] == '\n' {
			return 0
		}
	}
	return 0
}

// ScanLinkTitle scans a link title.
func ScanLinkTitle(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	delim := p[0]
	if delim != '"' && delim != '\'' && delim != '(' {
		return 0
	}
	closeDelim := delim
	if delim == '(' {
		closeDelim = ')'
	}
	i := 1
	for i < len(p) {
		if p[i] == '\\' && i+1 < len(p) && p[i+1] != 0 {
			i += 2
			continue
		}
		if p[i] == closeDelim {
			return i + 1
		}
		if p[i] == 0 {
			return 0
		}
		i++
	}
	return 0
}

// ScanSpaceChars scans consecutive space characters.
func ScanSpaceChars(p []byte) int {
	i := 0
	for i < len(p) && isSpacechar(p[i]) {
		i++
	}
	return i
}

// ScanATXHeadingStart scans an ATX heading start.
func ScanATXHeadingStart(p []byte) int {
	i := 0
	for i < len(p) && p[i] == '#' && i < 6 {
		i++
	}
	if i == 0 || i > 6 {
		return 0
	}
	if i < len(p) && (p[i] == ' ' || p[i] == '\t') {
		// skip trailing spaces
		j := i
		for j < len(p) && (p[j] == ' ' || p[j] == '\t') {
			j++
		}
		return j
	}
	if i < len(p) && (p[i] == '\r' || p[i] == '\n') {
		return i
	}
	return 0
}

// ScanSetextHeadingLine scans a setext heading underline.
// Returns 1 for =, 2 for -, 0 for no match.
func ScanSetextHeadingLine(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if c != '=' && c != '-' {
		return 0
	}
	i := 1
	for i < len(p) && p[i] == c {
		i++
	}
	for i < len(p) && (p[i] == ' ' || p[i] == '\t') {
		i++
	}
	if i < len(p) && (p[i] == '\r' || p[i] == '\n') {
		if c == '=' {
			return 1
		}
		return 2
	}
	return 0
}

// ScanOpenCodeFence scans an opening code fence.
func ScanOpenCodeFence(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if c != '`' && c != '~' {
		return 0
	}
	i := 1
	for i < len(p) && p[i] == c {
		i++
	}
	if i < 3 {
		return 0
	}
	// trailing context: [^`\r\n\x00]*[\r\n] for backticks,
	// [^\r\n\x00]*[\r\n] for tildes
	j := i
	for j < len(p) {
		if p[j] == '\r' || p[j] == '\n' {
			return i
		}
		if p[j] == 0 {
			return 0
		}
		if c == '`' && p[j] == '`' {
			return 0
		}
		j++
	}
	// If we hit end of input, it's still a valid fence start
	return i
}

// ScanCloseCodeFence scans a closing code fence.
func ScanCloseCodeFence(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	c := p[0]
	if c != '`' && c != '~' {
		return 0
	}
	i := 1
	for i < len(p) && p[i] == c {
		i++
	}
	if i < 3 {
		return 0
	}
	// trailing context: [ \t]*[\r\n]
	j := i
	for j < len(p) && (p[j] == ' ' || p[j] == '\t') {
		j++
	}
	if j < len(p) && (p[j] == '\r' || p[j] == '\n') {
		return i
	}
	return 0
}

// ScanDangerousURL scans for a dangerous URL scheme.
func ScanDangerousURL(p []byte) int {
	if bytes.HasPrefix(p, []byte("data:image/png")) ||
		bytes.HasPrefix(p, []byte("data:image/gif")) ||
		bytes.HasPrefix(p, []byte("data:image/jpeg")) ||
		bytes.HasPrefix(p, []byte("data:image/webp")) {
		return 0
	}
	for _, prefix := range []string{
		"javascript:", "vbscript:", "file:", "data:",
	} {
		if bytes.HasPrefix(p, []byte(prefix)) {
			return len(prefix)
		}
	}
	return 0
}
