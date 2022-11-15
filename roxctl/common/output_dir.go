package common

import (
	"os"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/errox"
)

type outputDirWrapper struct {
	OutputDir  *string
	defaultDir string
}

// NewOutputDir generates an output directory wrapper with a default for cobra
func NewOutputDir(s *string, defaultDir string) *outputDirWrapper {
	*s = defaultDir
	return &outputDirWrapper{
		OutputDir:  s,
		defaultDir: defaultDir,
	}
}

func (o *outputDirWrapper) String() string {
	return *o.OutputDir
}

func (o *outputDirWrapper) Set(input string) error {
	if input == "" {
		input = o.defaultDir
	}
	if _, err := os.Stat(input); err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to check status of directory %q", input)
	} else if err == nil {
		return errox.InvalidArgs.Newf("directory %q already exists. Please specify and new path to ensure expected results", input)
	}
	*o.OutputDir = input
	return nil
}

func (o *outputDirWrapper) Type() string {
	return "output directory"
}
