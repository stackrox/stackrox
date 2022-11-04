package gziputil

import (
	"bytes"
	"compress/gzip"
	"io"
)

// Decompress decompresses the given bytes of compressed data using gzip.
func Decompress(compressedData []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}

	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}

	return data, nil
}
