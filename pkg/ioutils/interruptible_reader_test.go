package ioutils

import (
	"io"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	currChunk []byte
	C         chan []byte
}

func newChunkReader() *chunkReader {
	return &chunkReader{
		C: make(chan []byte, 10),
	}
}

func (r *chunkReader) Read(buf []byte) (int, error) {
	for len(r.currChunk) == 0 {
		var ok bool
		r.currChunk, ok = <-r.C
		if !ok {
			return 0, io.EOF
		}
	}

	n := len(r.currChunk)
	if n > len(buf) {
		n = len(buf)
	}
	copy(buf, r.currChunk[:n])
	r.currChunk = r.currChunk[n:]
	return n, nil
}

func TestInterruptibleReader_InterruptPreRead(t *testing.T) {
	t.Parallel()

	cr := newChunkReader()

	ir, interrupt := NewInterruptibleReader(cr)
	cr.C <- []byte("foobar")

	buf := make([]byte, 3)
	n, err := io.ReadFull(ir, buf)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	assert.Equal(t, "foo", string(buf))

	interrupt()
	n, err = io.ReadFull(ir, buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interrupted")
	assert.Zero(t, n)
}

func TestInterruptibleReader_InterruptDuringRead(t *testing.T) {
	t.Parallel()

	cr := newChunkReader()

	ir, interrupt := NewInterruptibleReader(cr)
	var n int
	var err error
	done := concurrency.NewSignal()

	go func() {
		defer done.Signal()
		var buf [3]byte
		n, err = ir.Read(buf[:])
	}()

	time.Sleep(500 * time.Millisecond)
	assert.False(t, done.IsDone())
	interrupt()
	cr.C <- []byte("foobar")
	require.True(t, concurrency.WaitWithTimeout(&done, 1*time.Second))

	assert.Zero(t, n)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interrupted")
}
