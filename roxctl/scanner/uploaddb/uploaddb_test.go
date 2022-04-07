package uploaddb

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScannerUploadDbCommandFail(t *testing.T) {
	cmdNoFile := scannerUploadDbCommand{filename: "non-existing-filename"}

	actualErr := cmdNoFile.uploadDd()

	require.Error(t, actualErr)
	assert.ErrorIs(t, actualErr, fs.ErrNotExist)
}
