package ioutils

import (
	"io"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

type testReader struct {
	data     []byte
	closed   bool
	closeErr error
}

func newTestReader(data string) *testReader {
	return &testReader{
		data: []byte(data),
	}
}

func (r *testReader) get() io.Reader {
	return r
}

func (r *testReader) Read(buf []byte) (int, error) {
	if r.closed {
		return 0, errors.New("already closed")
	}
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	toRead := len(buf)
	if toRead > len(r.data) {
		toRead = len(r.data)
	}

	copy(buf[:toRead], r.data[:toRead])
	r.data = r.data[toRead:]
	return toRead, nil
}

func (r *testReader) Close() error {
	if r.closed {
		return errors.New("already closed")
	}
	r.closed = true
	return r.closeErr
}

func TestChainReadersFull(t *testing.T) {
	t.Parallel()

	r := ChainReadersEager(
		strings.NewReader("foo"),
		strings.NewReader("bar"),
		strings.NewReader("baz"))

	var buf [9]byte
	_, err := io.ReadFull(r, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foobarbaz", string(buf[:]))

	n, err := r.Read(buf[:])
	assert.Zero(t, n)
	assert.Equal(t, io.EOF, err)

	err = r.Close()
	assert.NoError(t, err)
}

func TestChainReadersEager_AllAreClosed(t *testing.T) {
	t.Parallel()

	trs := []*testReader{
		newTestReader("foo"),
		newTestReader("bar"),
		newTestReader("baz"),
	}
	r := ChainReadersEager(trs[0], trs[1], trs[2])

	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foob", string(buf[:]))

	assert.True(t, trs[0].closed)
	assert.False(t, trs[1].closed)
	assert.False(t, trs[2].closed)

	err = r.Close()
	assert.NoError(t, err)

	assert.True(t, trs[1].closed)
	assert.True(t, trs[2].closed)
}

func TestChainReadersLazy_FutureAreNotClosed(t *testing.T) {
	t.Parallel()

	trs := []*testReader{
		newTestReader("foo"),
		newTestReader("bar"),
		newTestReader("baz"),
	}
	r := ChainReadersLazy(trs[0].get, trs[1].get, trs[2].get)

	var buf [4]byte
	_, err := io.ReadFull(r, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foob", string(buf[:]))

	assert.True(t, trs[0].closed)
	assert.False(t, trs[1].closed)
	assert.False(t, trs[2].closed)

	err = r.Close()
	assert.NoError(t, err)

	assert.True(t, trs[1].closed)
	assert.False(t, trs[2].closed)
}

func TestChainReaders_CloseErrorPropagation(t *testing.T) {
	t.Parallel()

	trs := []*testReader{
		newTestReader("foo"),
		newTestReader("bar"),
	}

	idx := 0
	nextReader := func() io.Reader {
		if idx >= len(trs) {
			return nil
		}
		r := trs[idx]
		idx++
		return r
	}

	r := ChainReaders(nextReader, ChainReaderOpts{PropagateCloseErrors: true})
	tr0CloseErr := errors.New("error closing tr0")
	trs[0].closeErr = tr0CloseErr

	var buf [2]byte
	_, err := io.ReadFull(r, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "fo", string(buf[:]))

	n, err := io.ReadFull(r, buf[:])
	assert.Equal(t, 1, n)
	assert.Equal(t, "o", string(buf[:n]))
	assert.Equal(t, tr0CloseErr, err)
}
