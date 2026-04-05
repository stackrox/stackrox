package mtime_bench

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// hashWriteStat mirrors Go's internal cache hashWriteStat exactly
func hashWriteStat(h hash.Hash, info fs.FileInfo) {
	fmt.Fprintf(h, "stat %d %x %v %v\n", info.Size(), uint64(info.Mode()), info.ModTime(), info.IsDir())
}

// hashOpen mirrors Go's internal cache hashOpen (simplified)
func hashOpen(h hash.Hash, name string) {
	info, err := os.Stat(name)
	if err != nil {
		fmt.Fprintf(h, "err %v\n", err)
		return
	}
	hashWriteStat(h, info)
}

var timestamps = []struct {
	name   string
	touch  string
	goTime time.Time
}{
	{"epoch_zero", "197001010000", time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)},
	{"epoch_plus1d", "197001020000", time.Date(1970, 1, 2, 0, 0, 0, 0, time.UTC)},
	{"y2k_zeros", "200001010000", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)},
	{"baseline_2001", "200101010000", time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)},
	{"all_ones", "201111111111", time.Date(2011, 11, 11, 11, 11, 0, 0, time.UTC)},
	{"all_sevens", "200707070707", time.Date(2007, 7, 7, 7, 7, 0, 0, time.UTC)},
	{"max_digits", "199912312359", time.Date(1999, 12, 31, 23, 59, 0, 0, time.UTC)},
	{"recent", "202301150830", time.Date(2023, 1, 15, 8, 30, 0, 0, time.UTC)},
	{"future_max", "202512312359", time.Date(2025, 12, 31, 23, 59, 0, 0, time.UTC)},
	{"y1980", "198001010000", time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)},
	{"y1999", "199901010000", time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC)},
	{"y2026", "202601010000", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
}

// BenchmarkStatAndHash benchmarks the full os.Stat → fmt.Fprintf → sha256 pipeline
// per file, exactly as Go's test cache does it.
func BenchmarkStatAndHash(b *testing.B) {
	tmpDir := b.TempDir()

	for _, ts := range timestamps {
		files := make([]string, 50)
		for i := range files {
			f := filepath.Join(tmpDir, fmt.Sprintf("%s_%03d.go", ts.name, i))
			content := fmt.Sprintf("package foo\nvar x%d = %d\n", i, i)
			os.WriteFile(f, []byte(content), 0644)
			os.Chtimes(f, ts.goTime, ts.goTime)
			files[i] = f
		}

		b.Run(ts.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				h := sha256.New()
				for _, f := range files {
					hashOpen(h, f)
				}
				h.Sum(nil)
			}
		})
	}
}

// BenchmarkFormatOnly isolates just the fmt.Fprintf formatting of time.Time
// (no stat syscall)
func BenchmarkFormatOnly(b *testing.B) {
	for _, ts := range timestamps {
		info := fakeFileInfo{
			size:    1234,
			mode:    0644,
			modTime: ts.goTime,
			isDir:   false,
		}
		b.Run(ts.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				h := sha256.New()
				for range 50 {
					fmt.Fprintf(h, "stat %d %x %v %v\n", info.size, uint64(info.mode), info.modTime, info.isDir)
				}
				h.Sum(nil)
			}
		})
	}
}

// BenchmarkStatOnly isolates just the os.Stat syscall with different mtimes
func BenchmarkStatOnly(b *testing.B) {
	tmpDir := b.TempDir()

	for _, ts := range timestamps {
		files := make([]string, 50)
		for i := range files {
			f := filepath.Join(tmpDir, fmt.Sprintf("%s_%03d.go", ts.name, i))
			os.WriteFile(f, []byte("package foo\n"), 0644)
			os.Chtimes(f, ts.goTime, ts.goTime)
			files[i] = f
		}

		b.Run(ts.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				for _, f := range files {
					os.Stat(f)
				}
			}
		})
	}
}

// BenchmarkTimeFormat isolates just time.Time → string formatting
func BenchmarkTimeFormat(b *testing.B) {
	for _, ts := range timestamps {
		t := ts.goTime
		b.Run(ts.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				_ = fmt.Sprintf("%v", t)
			}
		})
	}
}

// BenchmarkTimeFormatBatch 300 calls (simulating 300 source files)
func BenchmarkTimeFormatBatch(b *testing.B) {
	for _, ts := range timestamps {
		t := ts.goTime
		b.Run(ts.name, func(b *testing.B) {
			b.ResetTimer()
			for b.Loop() {
				for range 300 {
					_ = fmt.Sprintf("%v", t)
				}
			}
		})
	}
}

type fakeFileInfo struct {
	size    int64
	mode    os.FileMode
	modTime time.Time
	isDir   bool
}

func (f fakeFileInfo) Name() string        { return "fake" }
func (f fakeFileInfo) Size() int64         { return f.size }
func (f fakeFileInfo) Mode() os.FileMode   { return f.mode }
func (f fakeFileInfo) ModTime() time.Time  { return f.modTime }
func (f fakeFileInfo) IsDir() bool         { return f.isDir }
func (f fakeFileInfo) Sys() any            { return nil }
