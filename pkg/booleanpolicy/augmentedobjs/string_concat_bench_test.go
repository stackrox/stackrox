package augmentedobjs

import (
	"strings"
	"testing"
)

const sep = "\t"

// Benchmark different string concatenation approaches
func BenchmarkStringConcat2Parts(b *testing.B) {
	instruction := "RUN"
	value := "apt-get update && apt-get install -y nginx"

	b.Run("fmt.Sprintf", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = instruction + sep + value
		}
	})

	b.Run("strings.Join", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = strings.Join([]string{instruction, value}, sep)
		}
	})

	b.Run("strings.Builder", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var sb strings.Builder
			sb.WriteString(instruction)
			sb.WriteString(sep)
			sb.WriteString(value)
			_ = sb.String()
		}
	})

	b.Run("strings.Builder+Grow", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var sb strings.Builder
			sb.Grow(len(instruction) + len(sep) + len(value))
			sb.WriteString(instruction)
			sb.WriteString(sep)
			sb.WriteString(value)
			_ = sb.String()
		}
	})

	b.Run("operator+", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = instruction + sep + value
		}
	})
}

func BenchmarkStringConcat3Parts(b *testing.B) {
	name := "nginx"
	version := "1.21.0-alpine"

	b.Run("strings.Join", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = strings.Join([]string{name, version}, sep)
		}
	})

	b.Run("strings.Builder", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var sb strings.Builder
			sb.WriteString(name)
			sb.WriteString(sep)
			sb.WriteString(version)
			_ = sb.String()
		}
	})

	b.Run("strings.Builder+Grow", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var sb strings.Builder
			sb.Grow(len(name) + len(sep) + len(version))
			sb.WriteString(name)
			sb.WriteString(sep)
			sb.WriteString(version)
			_ = sb.String()
		}
	})

	b.Run("operator+", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = name + sep + version
		}
	})
}

func BenchmarkStringConcat5Parts(b *testing.B) {
	source := "RAW"
	key := "DATABASE_URL"
	value := "postgres://user:pass@localhost:5432/db"

	b.Run("strings.Join", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = strings.Join([]string{source, key, value}, sep)
		}
	})

	b.Run("strings.Builder", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var sb strings.Builder
			sb.WriteString(source)
			sb.WriteString(sep)
			sb.WriteString(key)
			sb.WriteString(sep)
			sb.WriteString(value)
			_ = sb.String()
		}
	})

	b.Run("strings.Builder+Grow", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			var sb strings.Builder
			sb.Grow(len(source) + len(sep) + len(key) + len(sep) + len(value))
			sb.WriteString(source)
			sb.WriteString(sep)
			sb.WriteString(key)
			sb.WriteString(sep)
			sb.WriteString(value)
			_ = sb.String()
		}
	})

	b.Run("operator+", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			_ = source + sep + key + sep + value
		}
	})
}
