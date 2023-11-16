package uploaddb

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stackrox/rox/pkg/utils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	roxctlio "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func executeUpdateDBCommand(t *testing.T, serverURL string) (*bytes.Buffer, *bytes.Buffer, error) {
	tmpFile, errTempFile := os.CreateTemp("", "*.zip")
	if errTempFile != nil {
		return nil, nil, errTempFile
	}
	defer utils.IgnoreError(func() error {
		return os.Remove(tmpFile.Name())
	})

	testIO, _, stdOut, stdErr := roxctlio.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	cmd := Command(env)
	flags.AddConnectionFlags(cmd)
	flags.AddPassword(cmd)

	cmd.SilenceUsage = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// We are using common.DoHTTPRequestAndCheck200 inside uploadDd(). This
	// function uses  global variables that are set by command execution.
	// TODO(ROX-13638): Change uploadDB function to use HTTPClient from Environment.
	cmdArgs := []string{"--insecure-skip-tls-verify", "--insecure", "--endpoint", serverURL, "--password", "test"}
	cmdArgs = append(cmdArgs, "--scanner-db-file", tmpFile.Name())
	cmd.SetArgs(cmdArgs)

	cmdExecutionErr := cmd.Execute()
	if cmdExecutionErr != nil {
		return stdOut, stdErr, cmdExecutionErr
	}

	return stdOut, stdErr, nil
}

func TestScannerUploadDbCommand(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		cmdNoFile := scannerUploadDBCommand{filename: "non-existing-filename"}

		actualErr := cmdNoFile.uploadDB()

		require.Error(t, actualErr)
		assert.ErrorIs(t, actualErr, fs.ErrNotExist)
	})

	t.Run("server error", func(t *testing.T) {
		expectedErrorStr := "test-server-error"
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusBadRequest)
			_, _ = rw.Write([]byte(expectedErrorStr))
		}))
		defer server.Close()

		_, _, err := executeUpdateDBCommand(t, server.URL)

		require.Error(t, err)
		assert.Contains(t, err.Error(), expectedErrorStr)
	})

	t.Run("body read error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.Header().Set("Content-Length", "1")
		}))
		defer server.Close()

		stdOut, stdErr, err := executeUpdateDBCommand(t, server.URL)

		require.Error(t, err)
		require.NotNil(t, stdOut)
		require.NotNil(t, stdErr)

		assert.Empty(t, stdOut.String())
		assert.Empty(t, stdErr.String())
	})

	t.Run("scanner update-db", func(t *testing.T) {
		expectedResult := "test-roxctl-scanner-update-db"
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			_, _ = rw.Write([]byte(expectedResult))
		}))
		defer server.Close()

		stdOut, stdErr, err := executeUpdateDBCommand(t, server.URL)

		require.NoError(t, err)
		assert.Empty(t, stdErr.String())
		assert.Equal(t, fmt.Sprintln(expectedResult), stdOut.String())
	})
}
