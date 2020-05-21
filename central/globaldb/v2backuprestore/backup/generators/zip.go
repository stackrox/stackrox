package generators

import (
	"archive/tar"
	"archive/zip"
	"context"

	"github.com/pkg/errors"
)

// ZipGenerator writes a backup directly to a writer.
//go:generate mockgen-wrapper
type ZipGenerator interface {
	WriteTo(ctx context.Context, writer *zip.Writer) error
}

// PutStreamInZip calls the input Stream generator and outputs the results as a named file into a zip.
func PutStreamInZip(sGen StreamGenerator, fileNameInZip string) ZipGenerator {
	return &fromStreamToZip{
		sGen:     sGen,
		fileName: fileNameInZip,
	}
}

type fromStreamToZip struct {
	sGen     StreamGenerator
	fileName string
}

func (zgen *fromStreamToZip) WriteTo(ctx context.Context, writer *zip.Writer) error {
	subFile, err := writer.Create(zgen.fileName)
	if err != nil {
		return errors.Wrapf(err, "error creating %s in zip", zgen.fileName)
	}

	err = zgen.sGen.WriteTo(ctx, subFile)
	if err != nil {
		return errors.Wrapf(err, "unable to write %s to zip", zgen.fileName)
	}
	return nil
}

// PutTarInZip calls the input Stream generator and outputs the results as a named file into a zip.
func PutTarInZip(tGen TarGenerator, fileNameInZip string) ZipGenerator {
	return &fromTarToZip{
		tGen:     tGen,
		fileName: fileNameInZip,
	}
}

type fromTarToZip struct {
	tGen     TarGenerator
	fileName string
}

func (zgen *fromTarToZip) WriteTo(ctx context.Context, writer *zip.Writer) error {
	subFile, err := writer.Create(zgen.fileName)
	if err != nil {
		return errors.Wrapf(err, "error creating %s in zip", zgen.fileName)
	}
	tarWriter := tar.NewWriter(subFile)

	err = zgen.tGen.WriteTo(ctx, tarWriter)
	if err != nil {
		return errors.Wrapf(err, "unable to write %s to zip", zgen.fileName)
	}

	err = tarWriter.Close()
	if err != nil {
		return errors.Wrapf(err, "unable to close %s to zip", zgen.fileName)
	}
	return nil
}
