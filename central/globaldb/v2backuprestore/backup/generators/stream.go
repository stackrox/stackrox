package generators

import (
	"archive/zip"
	"context"
	"io"

	"github.com/pkg/errors"
)

// StreamGenerator writes a backup directly to a writer.
//go:generate mockgen-wrapper
type StreamGenerator interface {
	WriteTo(ctx context.Context, writer io.Writer) error
}

// PutZipInStream calls the input Zip generator and streams the output to an input writer.
func PutZipInStream(dGen ZipGenerator) StreamGenerator {
	return &fromZipToStream{
		zGen: dGen,
	}
}

type fromZipToStream struct {
	zGen ZipGenerator
}

func (sgen *fromZipToStream) WriteTo(ctx context.Context, writer io.Writer) error {
	zipeWriter := zip.NewWriter(writer)
	err := sgen.zGen.WriteTo(ctx, zipeWriter)
	if err != nil {
		return errors.Wrap(err, "unable to write to zip file")
	}
	return zipeWriter.Close()
}
