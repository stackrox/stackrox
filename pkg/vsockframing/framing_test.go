package vsockframing

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFraming(t *testing.T) {
	cases := map[string]struct {
		run func(t *testing.T)
	}{
		"should round-trip a payload": {
			run: func(t *testing.T) {
				payload := []byte("hello world")
				var buf bytes.Buffer
				require.NoError(t, WriteFrame(&buf, payload))
				got, err := ReadFrame(&buf, 1024)
				require.NoError(t, err)
				assert.Equal(t, payload, got)
			},
		},
		"should reject frame exceeding size limit": {
			run: func(t *testing.T) {
				payload := bytes.Repeat([]byte("x"), 100)
				var buf bytes.Buffer
				require.NoError(t, WriteFrame(&buf, payload))
				_, err := ReadFrame(&buf, 50)
				assert.ErrorContains(t, err, "exceeds limit")
			},
		},
		"should round-trip empty payload": {
			run: func(t *testing.T) {
				var buf bytes.Buffer
				require.NoError(t, WriteFrame(&buf, []byte{}))
				got, err := ReadFrame(&buf, 1024)
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		"should round-trip nil payload": {
			run: func(t *testing.T) {
				var buf bytes.Buffer
				require.NoError(t, WriteFrame(&buf, nil))
				got, err := ReadFrame(&buf, 1024)
				require.NoError(t, err)
				assert.Empty(t, got)
			},
		},
		"should error when length header is truncated": {
			run: func(t *testing.T) {
				_, err := ReadFrame(bytes.NewReader([]byte{0x00, 0x01}), 1024)
				assert.Error(t, err)
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			},
		},
		"should error when payload is truncated": {
			run: func(t *testing.T) {
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
