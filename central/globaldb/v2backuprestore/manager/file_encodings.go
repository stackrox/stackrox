package manager

import (
	"compress/flate"
	"io"

	"github.com/graph-gophers/graphql-go/errors"
	v1 "github.com/stackrox/stackrox/generated/api/v1"
)

var (
	decodingReaderFuncs = map[v1.DBExportManifest_EncodingType]func(io.Reader) io.Reader{
		v1.DBExportManifest_UNCOMPREESSED: func(r io.Reader) io.Reader { return r },
		v1.DBExportManifest_DEFLATED:      func(r io.Reader) io.Reader { return flate.NewReader(r) },
	}
)

func isSupportedFileEncoding(ty v1.DBExportManifest_EncodingType) bool {
	return decodingReaderFuncs[ty] != nil
}

func supportedFileEncodings() []v1.DBExportManifest_EncodingType {
	result := make([]v1.DBExportManifest_EncodingType, 0, len(decodingReaderFuncs))
	for ty := range decodingReaderFuncs {
		result = append(result, ty)
	}
	return result
}

func decodingReader(r io.Reader, encodingType v1.DBExportManifest_EncodingType) (io.Reader, error) {
	fn := decodingReaderFuncs[encodingType]
	if fn == nil {
		return nil, errors.Errorf("don't know how to decode %v", encodingType)
	}
	return fn(r), nil
}
