package cvss

import (
	"testing"

	"github.com/stackrox/rox/pkg/fileutils"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	defURL = "https://definitions.stackrox.io/e799c68a-671f-44db-9682-f24248cd0ffe/diff.zip"
)

func assertOnFileExistence(t *testing.T, path string, shouldExist bool) {
	exists, err := fileutils.Exists(path)
	require.NoError(t, err)
	assert.Equal(t, shouldExist, exists)
}
