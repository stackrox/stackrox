package generate

import (
	"os"

	"github.com/stackrox/rox/pkg/errox"
)

const defaultPath = "central-bundle"

type outputDirWrapper struct {
	OutputDir *string
}

func newOutputDir(s *string) *outputDirWrapper {
	*s = defaultPath
	return &outputDirWrapper{
		OutputDir: s,
	}
}

func (o *outputDirWrapper) String() string {
	return *o.OutputDir
}

func (o *outputDirWrapper) Set(input string) error {
	if input == "" {
		input = defaultPath
	}
	if _, err := os.Stat(input); err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		return errox.InvalidArgs.Newf("directory %q already exists. Please specify and new path to ensure expected results", input)
	}
	*o.OutputDir = input
	return nil
}

func (o *outputDirWrapper) Type() string {
	return "output directory"
}
