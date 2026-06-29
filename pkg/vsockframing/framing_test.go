package vsockframing

import (
	"bytes"
	"encoding/binary"
	"io"
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

func TestFramingEdgeCases(t *testing.T) {
	cases := map[string]struct {
		run func(t *testing.T)
	}{
		"should error when length header is truncated": {
			run: func(t *testing.T) {
				// Only 2 of the 4 length-prefix bytes, then EOF.
				_, err := ReadFrame(bytes.NewReader([]byte{0x00, 0x01}), 1024)
				assert.Error(t, err)
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			},
		},
		"should error when payload is truncated": {
			run: func(t *testing.T) {
				// Valid header claiming 100 bytes, but only 50 available.
				var buf bytes.Buffer
				_ = binary.Write(&buf, binary.BigEndian, uint32(100))
				buf.Write(make([]byte, 50))
				_, err := ReadFrame(&buf, 1024)
				assert.Error(t, err)
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			},
		},
		"should error when writer is closed": {
			run: func(t *testing.T) {
				r, w := io.Pipe()
				_ = w.Close()
				err := WriteFrame(w, []byte("data"))
				assert.Error(t, err)
				_ = r.Close()
			},
		},
	}
	for name, tc := range cases {
		t.Run(name, tc.run)
	}
}
