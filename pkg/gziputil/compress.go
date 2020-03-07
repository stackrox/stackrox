package gziputil

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
)

// Compress compresses the given bytes of raw data using gzip.
func Compress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Decompress decompresses the given bytes of compressed data using gzip.
func Decompress(compressedData []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}

	return data, nil
}
