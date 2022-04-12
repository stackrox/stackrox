package schema

import (
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	// SchemaGenFS holds the directory path used for postgres schema generation.
	SchemaGenFS = func() string {
		_, f, _, ok := runtime.Caller(0)
		if !ok {
			utils.Should(errors.New("failed to get postgres schema generation target directory"))
		}
		return filepath.Dir(f)
	}()
)
