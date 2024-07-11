package ioutils

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChanWriter_WriteSingle(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 1)
	w := NewChanWriter(context.Background(), ch)
	defer func() {
		assert.NoError(t, w.Close())
	}()

	n, err := w.Write([]byte("foo"))
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	chunk, ok := <-ch
	assert.True(t, ok)
	assert.Equal(t, "foo", string(chunk))

	n, err = w.Write([]byte("bar"))
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	chunk, ok = <-ch
	assert.True(t, ok)
	assert.Equal(t, "bar", string(chunk))
}

func TestChanWriter_WriteMultiNoAlias(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 2)
	w := NewChanWriter(context.Background(), ch)
	defer func() {
		assert.NoError(t, w.Close())
	}()

	buf := make([]byte, 3)

	copy(buf, []byte("foo"))
	n, err := w.Write(buf)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	copy(buf, []byte("bar"))
	n, err = w.Write(buf)
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	chunk, ok := <-ch
	assert.True(t, ok)
	assert.Equal(t, "foo", string(chunk))

	chunk, ok = <-ch
	assert.True(t, ok)
	assert.Equal(t, "bar", string(chunk))
}

func TestChanWriter_Close(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 1)
	w := NewChanWriter(context.Background(), ch)

	assert.NoError(t, w.Close())
	n, err := w.Write([]byte("foobar"))
	assert.Equal(t, ErrChanClosed, err)
	assert.Zero(t, n)

	_, ok := <-ch
	assert.False(t, ok)
}

func TestChanWriter_ContextError(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte)
	ctx, cancel := context.WithCancel(context.Background())
	w := NewChanWriter(ctx, ch)

	cancel()
	n, err := w.Write([]byte("foobar"))
	assert.Equal(t, context.Canceled, err)
	assert.Zero(t, n)

	assert.NoError(t, w.Close())

	// Test that the error remains stable, even if we now are additionally closed.
	for i := 0; i < 10; i++ {
		n, err := w.Write([]byte("foobar"))
		assert.Equal(t, context.Canceled, err)
		assert.Zero(t, n)
	}
}
