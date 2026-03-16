package buffer

// ContentSource provides byte-level access to original file content.
// Implementations: StringContent (heap), MmapContent (memory-mapped file).
type ContentSource interface {
	Slice(start, end int) []byte
	ByteAt(offset int) byte
	Len() int
	Close() error
}

// StringContent wraps a string for small files.
type StringContent struct {
	data string
}

func NewStringContent(s string) *StringContent {
	return &StringContent{data: s}
}

func (sc *StringContent) Slice(start, end int) []byte {
	return []byte(sc.data[start:end])
}

func (sc *StringContent) ByteAt(offset int) byte {
	return sc.data[offset]
}

func (sc *StringContent) Len() int {
	return len(sc.data)
}

func (sc *StringContent) Close() error {
	return nil
}
