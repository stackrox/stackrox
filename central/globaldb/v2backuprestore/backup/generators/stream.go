package generators

import (
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

// PutFileInStream generates stream from file and write to the output writer.
func PutFileInStream(filePath string) StreamGenerator {
	return &fromFileToStream{
		filePath: filePath,
	}
}

type fromFileToStream struct {
	filePath string
}

func (s *fromFileToStream) WriteTo(_ context.Context, writer io.Writer) error {
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
