package debug

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	io2 "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const timeoutWarningPrefix = "Timeout has been reached while creating diagnostic bundle"

func TestDownloadDiagnosticsTimeoutReached(t *testing.T) {
	shutdownServer := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		<-shutdownServer
	}))
	defer server.Close()
	defer close(shutdownServer)

	_, stdErr, err := executeDiagnosticsCommand(t, server.URL, time.Millisecond, "")

	require.Error(t, err)
	assert.True(t, isTimeoutError(err))
	assert.Contains(t, stdErr.String(), timeoutWarningPrefix)
}

func TestDownloadDiagnosticsServerError(t *testing.T) {
	expectedErrorStr := "test-server-error"
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusBadRequest)
		_, _ = rw.Write([]byte(expectedErrorStr))
	}))
	defer server.Close()

	_, stdErr, err := executeDiagnosticsCommand(t, server.URL, 20*time.Second, "")

	require.Error(t, err)
	assert.False(t, isTimeoutError(err))
	assert.Contains(t, err.Error(), expectedErrorStr)
	assert.NotContains(t, stdErr.String(), timeoutWarningPrefix)
}

func TestDownloadDiagnosticsSuccess(t *testing.T) {
	zipFilename := "test-file.zip"
	tempDir := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Header().Set("Content-Type", "application/zip")
		rw.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFilename))
		zipWriter := zip.NewWriter(rw)
		require.NoError(t, zipWriter.Close())
	}))
	defer server.Close()

	_, _, err := executeDiagnosticsCommand(t, server.URL, 20*time.Second, tempDir)

	assert.NoError(t, err)
	_, err = os.Stat(fmt.Sprintf("%s/%s", tempDir, zipFilename))
	assert.NoError(t, err)
}

func executeDiagnosticsCommand(t *testing.T, serverURL string, timeout time.Duration, dir string) (*bytes.Buffer, *bytes.Buffer, error) {
	testIO, _, stdOut, stdErr := io2.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	cmd := downloadDiagnosticsCommand(env)
	flags.AddConnectionFlags(cmd)
	flags.AddPassword(cmd)

	cmd.SilenceUsage = true
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)

	// We are using common.DoHTTPRequestAndCheck200 inside GetZip(). This
	// function uses  global variables that are set by command execution.
	// TODO(ROX-13638): Change GetZip function to use HTTPClient from Environment.
	cmdArgs := []string{"--insecure-skip-tls-verify", "--insecure",
		"--endpoint", serverURL,
		"--password", "test",
		"--timeout", timeout.String(),
		"--output-dir", dir,
	}
	cmd.SetArgs(cmdArgs)

	cmdExecutionErr := cmd.Execute()
	if cmdExecutionErr != nil {
		return stdOut, stdErr, cmdExecutionErr
	}

	return stdOut, stdErr, nil
}
