package generators

import (
	"archive/zip"
	"context"
	"io"
	"os"

	"github.com/pkg/errors"
)

// StreamGenerator writes a backup directly to a writer.
//
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

// PutFileInStream generates stream from file and write to the output writer.
func PutFileInStream(filePath string) StreamGenerator {
	return &fromFileToStream{
		filePath: filePath,
	}
}

type fromFileToStream struct {
	filePath string
}

func (s *fromFileToStream) WriteTo(ctx context.Context, writer io.Writer) error {
	file, err := os.Open(s.filePath)
	if err != nil {
		return errors.Wrapf(err, "could not open file %s", s.filePath)
	}
	_, err = io.Copy(writer, file)
	if err != nil {
		return errors.Wrapf(err, "could not copy file %s", s.filePath)
	}
	return nil
}
