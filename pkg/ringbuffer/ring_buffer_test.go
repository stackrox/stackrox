package ringbuffer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func concat(slices [][]byte) string {
	var result []byte
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return string(result)
}

func TestRingBuffer_NewInstance(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(37)
	assert.Equal(t, 37, rb.Capacity())
	assert.Zero(t, rb.Size())
	assert.Empty(t, rb.ReadAll())
	assert.Empty(t, rb.ReadFirst(5))
	assert.Empty(t, rb.ReadLast(5))
}

func TestRingBuffer_Read(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(4)
	rb.Write([]byte("foo"), nil)
	assert.Equal(t, "foo", concat(rb.ReadAll()))
	assert.Equal(t, "foo", concat(rb.ReadFirst(3)))
	assert.Equal(t, "foo", concat(rb.ReadLast(3)))
	assert.Equal(t, "o", concat(rb.Read(1, 1)))
	assert.Equal(t, "o", concat(rb.Read(-2, 1)))
	assert.Equal(t, "fo", concat(rb.ReadFirst(2)))
	assert.Equal(t, "oo", concat(rb.ReadLast(2)))

	rb.Write([]byte("bar"), nil)
	assert.Equal(t, "obar", concat(rb.ReadAll()))
	assert.Equal(t, "oba", concat(rb.ReadFirst(3)))
	assert.Equal(t, "ba", concat(rb.Read(1, 2)))
	assert.Equal(t, "ba", concat(rb.Read(-3, 2)))
	assert.Equal(t, "bar", concat(rb.ReadLast(3)))

	rb.Write([]byte("bazqux"), nil)
	assert.Equal(t, "zqux", concat(rb.ReadAll()))
	assert.Equal(t, "zqu", concat(rb.ReadFirst(3)))
	assert.Equal(t, "qux", concat(rb.ReadLast(3)))
	assert.Equal(t, "u", concat(rb.Read(2, 1)))
	assert.Equal(t, "q", concat(rb.Read(-3, 1)))
}

func TestRingBuffer_Write(t *testing.T) {
	t.Parallel()

	rb := NewRingBuffer(4)
	var evicted []byte
	cb := func(chunk []byte) {
		evicted = append(evicted, chunk...)
	}

	rb.Write([]byte("foo"), cb)
	assert.Empty(t, evicted)
	rb.Write([]byte("bar"), cb)
	assert.Equal(t, "fo", string(evicted))

	evicted = nil
	rb.Write([]byte("bazqux"), cb)
	assert.Equal(t, "obarba", string(evicted))
}
