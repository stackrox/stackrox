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
	return &fromTarToStream{
		dGen: dGen,
	}
}

type fromTarToStream struct {
	dGen DirectoryGenerator
}

func (tGen *fromTarToStream) WriteTo(ctx context.Context, writer *tar.Writer) error {
	path, err := tGen.dGen.WriteDirectory(ctx)
	if err != nil {
		return errors.Wrap(err, "unable to write to directory")
	}

	err = pkgTar.FromPath(path, writer)
	if err != nil {
		return errors.Wrap(err, "unable to tar directory")
	}

	return os.RemoveAll(path)
}
