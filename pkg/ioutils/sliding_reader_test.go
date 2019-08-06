package ioutils

import (
	"encoding/binary"
	"hash"
	"hash/crc32"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlidingReader_ReadAll(t *testing.T) {
	t.Parallel()

	rwr, err := NewSlidingReader(func() io.Reader { return strings.NewReader("foobarbazqux") }, 4, func() hash.Hash { return crc32.NewIEEE() })
	require.NoError(t, err)

	data, err := ioutil.ReadAll(rwr)
	assert.NoError(t, err)
	assert.Equal(t, "foobarbazqux", string(data))

	assert.NoError(t, rwr.Close())
}

func TestSlidingReader_RewindInBuffer(t *testing.T) {
	t.Parallel()

	readerCreations := 0
	rwr, err := NewSlidingReader(func() io.Reader {
		readerCreations++
		return ioutil.NopCloser(strings.NewReader("foobarbazqux"))
	}, 4, func() hash.Hash { return crc32.NewIEEE() })
	require.NoError(t, err)

	var buf [8]byte
	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foobarba", string(buf[:]))

	pos, err := rwr.Seek(4, io.SeekStart)
	assert.NoError(t, err)
	assert.EqualValues(t, 4, pos)
	assert.Equal(t, crc32.ChecksumIEEE([]byte("foob")), binary.BigEndian.Uint32(rwr.CurrentChecksum()))

	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "arbazqux", string(buf[:]))
	assert.NoError(t, rwr.Close())

	assert.Equal(t, 1, readerCreations)
}

func TestSlidingReader_RewindOutOfBuffer(t *testing.T) {
	t.Parallel()

	readerCreations := 0
	rwr, err := NewSlidingReader(func() io.Reader {
		readerCreations++
		return ioutil.NopCloser(strings.NewReader("foobarbazqux"))
	}, 4, func() hash.Hash { return crc32.NewIEEE() })
	require.NoError(t, err)

	var buf [8]byte
	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foobarba", string(buf[:]))

	pos, err := rwr.Seek(-5, io.SeekCurrent)
	assert.NoError(t, err)
	assert.EqualValues(t, 3, pos)
	assert.Equal(t, crc32.ChecksumIEEE([]byte("foo")), binary.BigEndian.Uint32(rwr.CurrentChecksum()))

	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "barbazqu", string(buf[:]))
	assert.NoError(t, rwr.Close())

	assert.Equal(t, 2, readerCreations)
}

func TestSlidingReader_RewindOutOfBuffer_WithSeeker(t *testing.T) {
	t.Parallel()

	readerCreations := 0
	rwr, err := NewSlidingReader(func() io.Reader {
		readerCreations++
		return strings.NewReader("foobarbazqux")
	}, 4, func() hash.Hash { return crc32.NewIEEE() })
	require.NoError(t, err)

	var buf [8]byte
	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foobarba", string(buf[:]))

	pos, err := rwr.Seek(3, io.SeekStart)
	assert.NoError(t, err)
	assert.EqualValues(t, 3, pos)
	assert.Equal(t, crc32.ChecksumIEEE([]byte("foo")), binary.BigEndian.Uint32(rwr.CurrentChecksum()))

	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "barbazqu", string(buf[:]))
	assert.NoError(t, rwr.Close())

	assert.Equal(t, 1, readerCreations)
}

func TestSlidingReader_FastForwardNear(t *testing.T) {
	t.Parallel()

	readerCreations := 0
	rwr, err := NewSlidingReader(func() io.Reader {
		readerCreations++
		return ioutil.NopCloser(strings.NewReader("foobarbazqux"))
	}, 4, func() hash.Hash { return crc32.NewIEEE() })
	require.NoError(t, err)

	var buf [4]byte
	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foob", string(buf[:]))

	pos, err := rwr.Seek(2, io.SeekCurrent)
	assert.NoError(t, err)
	assert.EqualValues(t, 6, pos)
	assert.Equal(t, crc32.ChecksumIEEE([]byte("foobar")), binary.BigEndian.Uint32(rwr.CurrentChecksum()))

	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "bazq", string(buf[:]))

	assert.NoError(t, rwr.Close())

	assert.Equal(t, 1, readerCreations)
}

func TestSlidingReader_FastForwardFar(t *testing.T) {
	t.Parallel()

	readerCreations := 0
	rwr, err := NewSlidingReader(func() io.Reader {
		readerCreations++
		return ioutil.NopCloser(strings.NewReader("foobarbazqux"))
	}, 4, func() hash.Hash { return crc32.NewIEEE() })
	require.NoError(t, err)

	var buf [3]byte
	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "foo", string(buf[:]))

	pos, err := rwr.Seek(8, io.SeekStart)
	assert.NoError(t, err)
	assert.EqualValues(t, 8, pos)
	assert.Equal(t, crc32.ChecksumIEEE([]byte("foobarba")), binary.BigEndian.Uint32(rwr.CurrentChecksum()))

	_, err = io.ReadFull(rwr, buf[:])
	assert.NoError(t, err)
	assert.Equal(t, "zqu", string(buf[:]))

	assert.NoError(t, rwr.Close())

	assert.Equal(t, 1, readerCreations)
}
