package ioutils

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/stackrox/pkg/concurrency"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContextBoundReader_ReadNormal(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("foobar")
	cbr := NewContextBoundReader(context.Background(), input)

	var buf [3]byte
	n, err := io.ReadFull(cbr, buf[:])
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(buf[:]))

	n, err = io.ReadFull(cbr, buf[:])
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "bar", string(buf[:]))
}

func TestContextBoundReader_ReadWithSequentialInterrupt(t *testing.T) {
	t.Parallel()

	input := strings.NewReader("foobar")
	ctx, cancel := context.WithCancel(context.Background())
	cbr := NewContextBoundReader(ctx, input)

	var buf [3]byte
	n, err := io.ReadFull(cbr, buf[:])
	assert.Equal(t, 3, n)
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(buf[:]))

	cancel()

	n, err = io.ReadFull(cbr, buf[:])
	assert.Zero(t, n)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}

func TestContextBoundReader_ReadWithParallelInterrupt(t *testing.T) {
	t.Parallel()

	cr := newChunkReader()

	ctx, cancel := context.WithCancel(context.Background())
	cbr := NewContextBoundReader(ctx, cr)

	var n int
	var err error
	done := concurrency.NewSignal()

	go func() {
		defer done.Signal()
		var buf [3]byte
		n, err = io.ReadFull(cbr, buf[:])
	}()

	time.Sleep(250 * time.Millisecond)
	cr.C <- []byte("fo")
	time.Sleep(250 * time.Millisecond)
	assert.False(t, done.IsDone())
	cancel()
	require.True(t, concurrency.WaitWithTimeout(&done, 1*time.Second))

	assert.Equal(t, 2, n)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "canceled")
}
