// Package bundle provides compression for production and fixture bundle writers.
package bundle

import (
	"errors"
	"io"

	"github.com/klauspost/compress/zstd"
)

// WriteCompressed serializes into a Zstandard stream and closes its encoder.
// The destination remains open. Serialization and encoder close errors are returned.
func WriteCompressed(w io.Writer, serialize func(io.Writer) error, opts ...zstd.EOption) error {
	enc, err := zstd.NewWriter(w, opts...)
	if err != nil {
		return err
	}
	writeErr := serialize(enc)
	return errors.Join(writeErr, enc.Close())
}
