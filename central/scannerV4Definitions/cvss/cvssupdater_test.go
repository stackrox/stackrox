package cvss

import (
	"testing"

	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	defURL = "https://storage.googleapis.com/scanner-v4-test/nvddata/"
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}
