package generators

import (
	"archive/zip"
	"context"
	"path"

	"github.com/pkg/errors"
)

// ZipGenerator writes a backup directly to a writer.
//
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

type fromPathMapToZip struct {
	pmGen    PathMapGenerator
	basePath string
}

// PutPathMapInZip gets the destination and source path from the path map and zips the files to destination in zip.
func PutPathMapInZip(pmGen PathMapGenerator, basePath string) ZipGenerator {
	return &fromPathMapToZip{
		pmGen:    pmGen,
		basePath: basePath,
	}
}

func (z *fromPathMapToZip) WriteTo(ctx context.Context, writer *zip.Writer) error {
	pathMap, err := z.pmGen.GeneratePathMap(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to generate path map")
	}
	for toPath, fromPath := range pathMap {
		stream := PutFileInStream(fromPath)
		err := PutStreamInZip(stream, path.Join(z.basePath, toPath)).WriteTo(ctx, writer)
		if err != nil {
			return errors.Wrapf(err, "error creating %s in zip", toPath)
		}
	}
	return nil
}
