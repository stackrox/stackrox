package ioutils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRWBuf_InMemLimit(t *testing.T) {
	b := NewRWBuf(10)
	n, err := b.Write([]byte("foobar"))
	require.NoError(t, err)
	assert.EqualValues(t, 6, n)
	assert.Nil(t, b.tmpFile)

	contents, size, err := b.Contents()
	require.NoError(t, err)
	assert.EqualValues(t, 6, size)

	buf := make([]byte, 4)
	n, err = contents.ReadAt(buf, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 4, n)
	assert.Equal(t, "ooba", string(buf))

	assert.NoError(t, b.Close())
}

func TestRWBuf_OutOfMemLimit_Immediately(t *testing.T) {
	b := NewRWBuf(4)
	n, err := b.Write([]byte("foobar"))
	require.NoError(t, err)
	assert.EqualValues(t, 6, n)
	assert.NotNil(t, b.tmpFile)

	contents, size, err := b.Contents()
	require.NoError(t, err)
	assert.EqualValues(t, 6, size)

	buf := make([]byte, 4)
	n, err = contents.ReadAt(buf, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 4, n)
	assert.Equal(t, "ooba", string(buf))

	assert.NoError(t, b.Close())
}

func TestRWBuf_OutOfMemLimit_AfterOneWrite(t *testing.T) {
	b := NewRWBuf(4)
	n, err := b.Write([]byte("foo"))
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
	assert.Nil(t, b.tmpFile)

	n, err = b.Write([]byte("bar"))
	require.NoError(t, err)
	assert.EqualValues(t, 3, n)
	assert.NotNil(t, b.tmpFile)

	contents, size, err := b.Contents()
	require.NoError(t, err)
	assert.EqualValues(t, 6, size)

	buf := make([]byte, 4)
	n, err = contents.ReadAt(buf, 1)
	require.NoError(t, err)
	assert.EqualValues(t, 4, n)
	assert.Equal(t, "ooba", string(buf))

	assert.NoError(t, b.Close())
}
