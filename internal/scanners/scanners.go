// Package scanners provides hand-rolled byte-level pattern scanners.
// These replace the re2c-generated C scanners for performance in Go.
package scanners

// ScanScheme scans an autolink scheme at the start of p.
// Returns the number of bytes consumed, or 0 if no match.
func ScanScheme(p []byte) int {
	// TODO: stub
	return 0
}

// ScanAutolinkURI scans an autolink URI.
func ScanAutolinkURI(p []byte) int {
	// TODO: stub
	return 0
}

// ScanAutolinkEmail scans an autolink email.
func ScanAutolinkEmail(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLTag scans an HTML tag.
func ScanHTMLTag(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLComment scans an HTML comment.
func ScanHTMLComment(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLPI scans an HTML processing instruction.
func ScanHTMLPI(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLDeclaration scans an HTML declaration.
func ScanHTMLDeclaration(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLCDATA scans a CDATA section.
func ScanHTMLCDATA(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockStart scans for the start of an HTML block (types 1-6).
func ScanHTMLBlockStart(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockStart7 scans for the start of HTML block type 7.
func ScanHTMLBlockStart7(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockEnd1 scans for the end of HTML block type 1.
func ScanHTMLBlockEnd1(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockEnd2 scans for the end of HTML block type 2.
func ScanHTMLBlockEnd2(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockEnd3 scans for the end of HTML block type 3.
func ScanHTMLBlockEnd3(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockEnd4 scans for the end of HTML block type 4.
func ScanHTMLBlockEnd4(p []byte) int {
	// TODO: stub
	return 0
}

// ScanHTMLBlockEnd5 scans for the end of HTML block type 5.
func ScanHTMLBlockEnd5(p []byte) int {
	// TODO: stub
	return 0
}

// ScanLinkTitle scans a link title.
func ScanLinkTitle(p []byte) int {
	// TODO: stub
	return 0
}

// ScanSpaceChars scans consecutive space characters.
func ScanSpaceChars(p []byte) int {
	// TODO: stub
	return 0
}

// ScanATXHeadingStart scans an ATX heading start.
func ScanATXHeadingStart(p []byte) int {
	// TODO: stub
	return 0
}

// ScanSetextHeadingLine scans a setext heading underline.
func ScanSetextHeadingLine(p []byte) int {
	// TODO: stub
	return 0
}

// ScanOpenCodeFence scans an opening code fence.
func ScanOpenCodeFence(p []byte) int {
	// TODO: stub
	return 0
}

// ScanCloseCodeFence scans a closing code fence.
func ScanCloseCodeFence(p []byte) int {
	// TODO: stub
	return 0
}

// ScanDangerousURL scans for a dangerous URL scheme.
func ScanDangerousURL(p []byte) int {
	// TODO: stub
	return 0
}
