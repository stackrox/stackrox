package bundle

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
)

type destination struct {
	bytes.Buffer
	closed bool
}

func (d *destination) Close() error { d.closed = true; return nil }

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, io.ErrClosedPipe }

func TestWriteCompressed(t *testing.T) {
	var dst destination
	require.NoError(t, WriteCompressed(&dst, func(w io.Writer) error {
		_, err := io.WriteString(w, "complete payload <>&")
		return err
	}))
	require.False(t, dst.closed)
	dec, err := zstd.NewReader(bytes.NewReader(dst.Bytes()))
	require.NoError(t, err)
	defer dec.Close()
	got, err := io.ReadAll(dec)
	require.NoError(t, err)
	require.Equal(t, "complete payload <>&", string(got))
}

func TestWriteCompressedErrors(t *testing.T) {
	serializationErr := errors.New("serialization failed")
	for name, dst := range map[string]io.Writer{
		"serialization":           &bytes.Buffer{},
		"serialization and close": failingWriter{},
	} {
		t.Run(name, func(t *testing.T) {
			err := WriteCompressed(dst, func(w io.Writer) error {
				_, err := io.WriteString(w, "buffered payload")
				require.NoError(t, err)
				return serializationErr
			})
			require.ErrorIs(t, err, serializationErr)
			if name == "serialization and close" {
				require.ErrorIs(t, err, io.ErrClosedPipe)
			}
		})
	}
	require.ErrorIs(t, WriteCompressed(failingWriter{}, func(w io.Writer) error {
		_, err := io.WriteString(w, "payload")
		return err
	}), io.ErrClosedPipe)
	require.Error(t, WriteCompressed(&bytes.Buffer{}, func(io.Writer) error {
		t.Fatal("serializer called with invalid encoder options")
		return nil
	}, zstd.WithEncoderConcurrency(-1)))
}
