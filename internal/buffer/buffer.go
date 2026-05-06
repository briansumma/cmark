// Package buffer provides a growable byte buffer for the parser.
package buffer

// Buffer is a growable byte buffer.
type Buffer struct {
	data []byte
}

// New creates a new buffer with the given initial capacity.
func New(capacity int) *Buffer {
	return &Buffer{data: make([]byte, 0, capacity)}
}

// Grow ensures the buffer can hold at least targetSize bytes.
func (b *Buffer) Grow(targetSize int) {
	if cap(b.data) < targetSize {
		newData := make([]byte, len(b.data), targetSize)
		copy(newData, b.data)
		b.data = newData
	}
}

// Set replaces the buffer contents with data.
func (b *Buffer) Set(data []byte) {
	b.data = append(b.data[:0], data...)
}

// AppendByte appends a single byte.
func (b *Buffer) AppendByte(c byte) {
	b.data = append(b.data, c)
}

// Append appends data.
func (b *Buffer) Append(data []byte) {
	b.data = append(b.data, data...)
}

// AppendString appends a string.
func (b *Buffer) AppendString(s string) {
	b.data = append(b.data, s...)
}

// Clear empties the buffer.
func (b *Buffer) Clear() {
	b.data = b.data[:0]
}

// Drop removes the first n bytes.
func (b *Buffer) Drop(n int) {
	if n >= len(b.data) {
		b.data = b.data[:0]
		return
	}
	b.data = b.data[n:]
}

// Truncate sets the buffer length to len.
func (b *Buffer) Truncate(length int) {
	if length < len(b.data) {
		b.data = b.data[:length]
	}
}

// RTrim removes trailing whitespace.
func (b *Buffer) RTrim() {
	// TODO: stub
}

// Trim removes leading and trailing whitespace.
func (b *Buffer) Trim() {
	// TODO: stub
}

// NormalizeWhitespace collapses consecutive whitespace.
func (b *Buffer) NormalizeWhitespace() {
	// TODO: stub
}

// Unescape removes backslash escapes.
func (b *Buffer) Unescape() {
	// TODO: stub
}

// Bytes returns the buffer contents.
func (b *Buffer) Bytes() []byte {
	return b.data
}

// String returns the buffer contents as a string.
func (b *Buffer) String() string {
	return string(b.data)
}

// Len returns the length of the buffer.
func (b *Buffer) Len() int {
	return len(b.data)
}

// Detach returns the underlying slice and resets the buffer.
func (b *Buffer) Detach() []byte {
	data := b.data
	b.data = nil
	return data
}
