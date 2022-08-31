package generators

import (
	"archive/zip"
	"context"
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

// FileGenerator is a generator that produces a backup in the form of a file.
//
//go:generate mockgen-wrapper
type FileGenerator interface {
	WriteFile(ctx context.Context, path string) error
}

// PutStreamInFile calls a StreamGenerator and outputs the results to a file.
func PutStreamInFile(sgen StreamGenerator) FileGenerator {
	return &fromStreamToFile{sgen: sgen}
}

type fromStreamToFile struct {
	sgen StreamGenerator
}

func (fgen *fromStreamToFile) WriteFile(ctx context.Context, path string) error {
	// Get a Buffer to Write To
	outFile, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "unable to create path for file %s", path)
	}
	defer utils.IgnoreError(outFile.Close)

	return fgen.sgen.WriteTo(ctx, outFile)
}

// PutZipInFile calls a ZipGenerator and outputs the results to a file.
func PutZipInFile(sgen ZipGenerator) FileGenerator {
	return &fromZipToFile{sgen: sgen}
}

type fromZipToFile struct {
	sgen ZipGenerator
}

func (fgen *fromZipToFile) WriteFile(ctx context.Context, path string) error {
	// Get a Buffer to Write To
	outFile, err := os.Create(path)
	if err != nil {
		return errors.Wrapf(err, "unable to create path for file %s", path)
	}
	defer utils.IgnoreError(outFile.Close)

	zipWriter := zip.NewWriter(outFile)
	err = fgen.sgen.WriteTo(ctx, zipWriter)
	if err != nil {
		return errors.Wrap(err, "unable to write to zip file")
	}
	return zipWriter.Close()
}
