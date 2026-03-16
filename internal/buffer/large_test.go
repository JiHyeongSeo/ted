package buffer

import (
	"os"
	"testing"
)

func BenchmarkOpenLargeFile(b *testing.B) {
	// Create a ~50MB temp file
	f, err := os.CreateTemp("", "ted-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())

	line := "Benchmark line content for testing large file performance.\n"
	for i := 0; i < 900000; i++ {
		f.WriteString(line)
	}
	f.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf, err := OpenFile(f.Name())
		if err != nil {
			b.Fatal(err)
		}
		buf.Close()
	}
}

func BenchmarkLineAccess(b *testing.B) {
	f, err := os.CreateTemp("", "ted-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	defer os.Remove(f.Name())

	line := "Benchmark line content for testing.\n"
	for i := 0; i < 900000; i++ {
		f.WriteString(line)
	}
	f.Close()

	buf, err := OpenFile(f.Name())
	if err != nil {
		b.Fatal(err)
	}
	defer buf.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buf.Line(i % buf.LineCount())
	}
}
