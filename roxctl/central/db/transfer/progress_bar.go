package transfer

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"
)

// ProgressBar tracks byte progress and periodically logs a status line.
type ProgressBar struct {
	current atomic.Int64
	total   int64
	name    string
	out     io.Writer
	done    chan struct{}
}

// CreateProgressBar creates a progress tracker that prints periodic status lines.
func CreateProgressBar(ctx context.Context, name string, totalSize int64, out io.Writer) (*ProgressBar, func()) {
	if out == nil {
		out = os.Stderr //nolint:forbidigo // TODO(ROX-13473)
	}

	bar := &ProgressBar{
		total: totalSize,
		name:  name,
		out:   out,
		done:  make(chan struct{}),
	}

	go bar.render(ctx)

	shutdown := func() {
		close(bar.done)
		bar.printLine()
	}
	return bar, shutdown
}

func (b *ProgressBar) IncrBy(n int) {
	b.current.Add(int64(n))
}

func (b *ProgressBar) Current() int64 {
	return b.current.Load()
}

func (b *ProgressBar) SetCurrent(n int64) {
	b.current.Store(n)
}

func (b *ProgressBar) ProxyReader(r io.ReadCloser) io.ReadCloser {
	return &countingReader{r: r, bar: b}
}

func (b *ProgressBar) render(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			b.printLine()
		case <-b.done:
			return
		case <-ctx.Done():
			return
		}
	}
}

func (b *ProgressBar) printLine() {
	cur := b.current.Load()
	if b.total > 0 {
		pct := float64(cur) / float64(b.total) * 100
		fmt.Fprintf(b.out, "%s: %s / %s (%.0f%%)\n", b.name, FormatSize(cur), FormatSize(b.total), pct)
	} else {
		fmt.Fprintf(b.out, "%s: %s\n", b.name, FormatSize(cur))
	}
}

// FormatSize formats a byte count as a human-readable string (KiB, MiB, GiB).
func FormatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMG"[exp])
}

type countingReader struct {
	r   io.ReadCloser
	bar *ProgressBar
}

func (cr *countingReader) Read(p []byte) (int, error) {
	n, err := cr.r.Read(p)
	if n > 0 {
		cr.bar.IncrBy(n)
	}
	return n, err
}

func (cr *countingReader) Close() error {
	return cr.r.Close()
}
