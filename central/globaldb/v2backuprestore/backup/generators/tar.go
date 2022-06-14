package generators

import (
	"archive/tar"
	"context"
	"os"

	"github.com/pkg/errors"
	pkgTar "github.com/stackrox/rox/pkg/tar"
)

// TarGenerator writes a backup directly to a writer.
//go:generate mockgen-wrapper
type TarGenerator interface {
	WriteTo(ctx context.Context, writer *tar.Writer) error
}

// PutDirectoryInTar calls the input Directory generator on the input temporary data path, and outpus the results to a tar.
func PutDirectoryInTar(dGen DirectoryGenerator) TarGenerator {
	return &fromDirectoryToTarStream{
		dGen: dGen,
	}
}

type fromDirectoryToTarStream struct {
	dGen DirectoryGenerator
}

func (t *fromDirectoryToTarStream) WriteTo(ctx context.Context, writer *tar.Writer) error {
	path, err := t.dGen.WriteDirectory(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to write to directory")
	}

	err = pkgTar.FromPath(path, writer)
	if err != nil {
		return errors.Wrap(err, "unable to tar directory")
	}

	return os.RemoveAll(path)
}

type fromPathsToTarStream struct {
	pmGen PathMapGenerator
}

// PutPathsInTar creates tar from a map of structured paths in tar to its source path.
func PutPathsInTar(pmGen PathMapGenerator) TarGenerator {
	return &fromPathsToTarStream{
		pmGen: pmGen,
	}
}

func (t *fromPathsToTarStream) WriteTo(ctx context.Context, writer *tar.Writer) error {
	pathMap, err := t.pmGen.GeneratePathMap(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to generate path map")
	}
	if err = pkgTar.FromPathMap(pathMap, writer); err != nil {
		return errors.Wrap(err, "unable to tar path map")
	}
	return nil
}
