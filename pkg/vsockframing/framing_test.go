package vsockframing

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteAndReadFrame(t *testing.T) {
	payload := []byte("hello world")
	var buf bytes.Buffer

	err := WriteFrame(&buf, payload)
	require.NoError(t, err)

	got, err := ReadFrame(&buf, 1024)
	require.NoError(t, err)
	assert.Equal(t, payload, got)
}

func TestReadFrame_ExceedsLimit(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), 100)
	var buf bytes.Buffer

	err := WriteFrame(&buf, payload)
	require.NoError(t, err)

	_, err = ReadFrame(&buf, 50)
	assert.ErrorContains(t, err, "exceeds limit")
}

func TestReadFrame_EmptyPayload(t *testing.T) {
	var buf bytes.Buffer
	err := WriteFrame(&buf, []byte{})
	require.NoError(t, err)

	got, err := ReadFrame(&buf, 1024)
	require.NoError(t, err)
	assert.Empty(t, got)
}
