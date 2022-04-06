package uploaddb

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScannerUploadDbCommandFail(t *testing.T) {
	t.Run("non existing filename", func(t *testing.T) {
		cmdNoFile := scannerUploadDbCommand{filename: "non-existing-filename"}

		require.EqualError(t, cmdNoFile.uploadDd(), "could not open file: open non-existing-filename: no such file or directory")
	})
}
