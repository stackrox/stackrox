package ioutils

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChanReader_ReadSingle(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 1)
	r := NewChanReader(context.Background(), ch)
	ch <- []byte("foo")

	var buf [3]byte
	n, err := r.Read(buf[:])
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "foo", string(buf[:]))

	ch <- []byte("ba")
	n, err = r.Read(buf[:])
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, "ba", string(buf[:2]))
}

func TestChanReader_ReadChunked(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 1)
	r := NewChanReader(context.Background(), ch)
	ch <- []byte("foobar")

	var buf [3]byte
	n, err := r.Read(buf[:])
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "foo", string(buf[:]))

	n, err = r.Read(buf[:])
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, "bar", string(buf[:]))
}

func TestChanReader_ReadBatched(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 2)
	r := NewChanReader(context.Background(), ch)
	ch <- []byte("foo")
	ch <- []byte("bar")

	var buf [4]byte
	n, err := io.ReadFull(r, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "foob", string(buf[:]))

	n, err = r.Read(buf[:])
	assert.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.Equal(t, "ar", string(buf[:2]))
}

func TestChanReader_ReadClosed(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 1)
	r := NewChanReader(context.Background(), ch)
	close(ch)

	var buf [4]byte
	n, err := r.Read(buf[:])
	assert.Zero(t, n)
	assert.Equal(t, io.EOF, err)
}

func TestChanReader_ReadError(t *testing.T) {
	t.Parallel()

	ch := make(chan []byte, 1)
	ctx, cancel := context.WithCancel(context.Background())
	r := NewChanReader(ctx, ch)
	cancel()

	var buf [4]byte
	n, err := r.Read(buf[:])
	assert.Zero(t, n)
	assert.Equal(t, context.Canceled, err)

	// Test that the error reemains stable, even if there is new data to read.
	ch <- []byte("foo")
	for i := 0; i < 10; i++ {
		n, err := r.Read(buf[:])
		assert.Zero(t, n)
		assert.Equal(t, context.Canceled, err)
	}
}
