package binenc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestByteSliceList(t *testing.T) {
	input := [][]byte{
		[]byte("foobar\x00baz"),
		[]byte("\x00\x01\x00"),
	}

	encoded := EncodeBytesList(input...)
	decoded, err := DecodeBytesList(encoded)
	require.NoError(t, err)

	require.Equal(t, len(input), len(decoded))

	for i, val := range input {
		assert.Equal(t, string(val), string(decoded[i]))
	}
}
