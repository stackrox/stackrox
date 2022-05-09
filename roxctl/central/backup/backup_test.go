package backup

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFilePath(t *testing.T) {
	tempDir := t.TempDir()
	const existingFileName = "existing.file"
	existingFile, err := os.Create(filepath.Join(tempDir, existingFileName))
	require.NoError(t, err)
	defer utils.IgnoreError(existingFile.Close)

	testCases := []struct {
		description string

		serverProvidedFileName string
		userProvidedOutput     string

		expectedPath string
		errExpected  bool
	}{
		{
			description:            "No user provided output",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     "",

			expectedPath: "stackrox.db",
		},
		{
			description:            "User provides existing directory",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     tempDir,

			expectedPath: filepath.Join(tempDir, "stackrox.db"),
		},
		{
			description:            "User provides file in non existent directory",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     "NONEXISTENTPARENTDIR/test.db",

			errExpected: true,
		},
		{
			description:            "User provides non-existent filename in existing directory",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     filepath.Join(tempDir, "nonexisting"),

			expectedPath: filepath.Join(tempDir, "nonexisting"),
		},
		{
			description:            "Use provides existing filepath",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     filepath.Join(tempDir, existingFileName),

			expectedPath: filepath.Join(tempDir, existingFileName),
		},
		{
			description:            "User provides existing directory with a trailing /",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     stringutils.EnsureSuffix(tempDir, string(os.PathSeparator)),

			expectedPath: filepath.Join(tempDir, "stackrox.db"),
		},
		{
			description:            "User tries to treat a file as a directory",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     filepath.Join(tempDir, existingFileName, "existingfilecanthaveafilewithinit"),

			errExpected: true,
		},
		{
			description:            "User provides non existing directory with a trailing /",
			serverProvidedFileName: "stackrox.db",
			userProvidedOutput:     stringutils.EnsureSuffix(filepath.Join(tempDir, "nonexisting"), string(os.PathSeparator)),

			errExpected: true,
		},
	}

	for _, testCase := range testCases {
		c := testCase
		t.Run(c.description, func(t *testing.T) {
			header := http.Header{
				"Content-Disposition": []string{fmt.Sprintf("attachment; filename=\"%s\"", c.serverProvidedFileName)},
			}
			gotPath, err := getFilePath(header, c.userProvidedOutput)
			if c.errExpected {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, c.expectedPath, gotPath)
		})
	}
}
