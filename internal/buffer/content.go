package buffer

import (
	"fmt"
	"os"
	"syscall"
)

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

// MmapContent provides memory-mapped file access for large files.
type MmapContent struct {
	data []byte
	size int
}

func NewMmapContent(path string) (*MmapContent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := int(info.Size())
	if size == 0 {
		return &MmapContent{data: nil, size: 0}, nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, size,
		syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %w", err)
	}

	return &MmapContent{data: data, size: size}, nil
}

func (mc *MmapContent) Slice(start, end int) []byte {
	if mc.data == nil {
		return nil
	}
	out := make([]byte, end-start)
	copy(out, mc.data[start:end])
	return out
}

func (mc *MmapContent) ByteAt(offset int) byte {
	return mc.data[offset]
}

func (mc *MmapContent) Len() int {
	return mc.size
}

func (mc *MmapContent) Close() error {
	if mc.data != nil {
		return syscall.Munmap(mc.data)
	}
	return nil
}
