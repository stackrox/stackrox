package uploaddb

import (
	"io/fs"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerUploadDbCommandFail(t *testing.T) {
	t.Run("non existing filename", func(t *testing.T) {
		cmdNoFile := scannerUploadDbCommand{filename: "non-existing-filename"}

		actualErr := cmdNoFile.uploadDd()

		require.Error(t, actualErr)
		assert.ErrorIs(t, errors.Cause(actualErr), fs.ErrNotExist)
		assert.EqualError(t, cmdNoFile.uploadDd(), "could not open file: open non-existing-filename: no such file or directory")
	})
}
