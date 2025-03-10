package downloaddb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/pkg/version/testutils"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	roxctlio "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createCentralServer(version string, retErr bool) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if retErr {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		md := &v1.Metadata{}
		md.Version = version
		data, err := json.Marshal(md)
		if err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, _ = rw.Write(data)
	}))

	return server
}

func TestBuildBundleFileNames(t *testing.T) {
	testIO, _, _, _ := roxctlio.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	// For readability.
	skipVariants := true
	wantErr := true
	tcs := []struct {
		version      string
		skipVariants bool
		wantErr      bool
		want         []string
	}{
		// Invalid
		{"4.v3.0", !skipVariants, wantErr, nil},

		// Prior to Scanner V4.
		{"4.3.0", !skipVariants, !wantErr, []string{
			"scanner-vuln-updates.zip",
		}},
		{"4.3.0", skipVariants, !wantErr, []string{
			"scanner-vuln-updates.zip",
		}},

		// Post Scanner V4.
		{"4.4.0", !skipVariants, !wantErr, []string{
			"4.4.0/scanner-vulns-4.4.0.zip",
			"4.4/scanner-vulns-4.4.zip",
		}},
		{"4.4.0", skipVariants, !wantErr, []string{
			"4.4.0/scanner-vulns-4.4.0.zip",
		}},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("%s-%t", tc.version, tc.skipVariants), func(t *testing.T) {
			cmd := &scannerDownloadDBCommand{
				env:          env,
				version:      tc.version,
				skipVariants: tc.skipVariants,
			}

			got, err := cmd.buildBundleFileNames()
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}
func TestDetectVersion(t *testing.T) {
	testIO, _, _, _ := roxctlio.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	t.Setenv("ROX_ADMIN_PASSWORD", "fake")
	t.Setenv("ROX_INSECURE_CLIENT", "true")
	flags.AddCentralConnectionFlags(&cobra.Command{}) // init flags.passwordChanged to avoid nil pointer
	_, _, _ = flags.EndpointAndPlaintextSetting()     // init endpoint

	t.Run("use version from flag", func(t *testing.T) {
		cmd := &scannerDownloadDBCommand{env: env, version: "1.2.3"}
		got := cmd.detectVersion()
		assert.Equal(t, "1.2.3", got)
	})

	t.Run("use version from Central", func(t *testing.T) {
		server := createCentralServer("3.2.1", false)
		defer server.Close()
		t.Setenv("ROX_ENDPOINT", server.URL)
		t.Setenv("ROX_CLIENT_MAX_RETRIES", "0")

		cmd := &scannerDownloadDBCommand{env: env}
		got := cmd.detectVersion()
		assert.Equal(t, "3.2.1", got)
	})

	t.Run("use version embedded in roxctl if Central fails", func(t *testing.T) {
		server := createCentralServer("", true)
		defer server.Close()
		t.Setenv("ROX_ENDPOINT", server.URL)
		t.Setenv("ROX_CLIENT_MAX_RETRIES", "0")

		cmd := &scannerDownloadDBCommand{env: env}
		testutils.SetMainVersion(t, "4.3.2")

		got := cmd.detectVersion()
		assert.Equal(t, "4.3.2", got)
	})

	t.Run("use version embedded in roxctl", func(t *testing.T) {
		cmd := &scannerDownloadDBCommand{env: env, skipCentral: true}
		testutils.SetMainVersion(t, "4.5.6")

		got := cmd.detectVersion()
		assert.NotEmpty(t, got)
	})
}

func TestBuildAndValidateOutputFileName(t *testing.T) {
	t.Run("use as input if no flag", func(t *testing.T) {
		cmd := &scannerDownloadDBCommand{force: true}
		got, err := cmd.buildAndValidateOutputFileName("filename")
		require.NoError(t, err)
		assert.Equal(t, "filename", got)
	})

	t.Run("use flag if provided", func(t *testing.T) {
		cmd := &scannerDownloadDBCommand{force: true, filename: "flag"}
		got, err := cmd.buildAndValidateOutputFileName("filename")
		require.NoError(t, err)
		assert.Equal(t, "flag", got)
	})

	t.Run("fail if file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		filename := filepath.Join(tmpDir, "filename.txt")
		f, err := os.Create(filename)
		require.NoError(t, err)
		_ = f.Close()

		cmd := &scannerDownloadDBCommand{}
		_, err = cmd.buildAndValidateOutputFileName(filename)
		require.Error(t, err)
	})
}

func serveRawMetadata(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/v1/metadata", r.URL.Path)

		responsePayload := `{
	"version": "4.3.2.1",
	"buildFlavor": "unitTest",
	"releaseBuild": false,
	"licenseStatus": "VALID",
	"dummyTestField": "to test backward/forward compatibility"
}`
		_, _ = w.Write([]byte(responsePayload))
	}
}

func TestVersionFromCentral(t *testing.T) {
	server := httptest.NewServer(serveRawMetadata(t))
	defer server.Close()

	// Required for picking up the endpoint used by GetRoxctlHTTPClient. Currently, it is not possible to inject this
	// otherwise.
	t.Setenv("ROX_ENDPOINT", server.URL)

	t.Setenv("ROX_ADMIN_PASSWORD", "fake")
	t.Setenv("ROX_INSECURE_CLIENT", "true")
	flags.AddCentralConnectionFlags(&cobra.Command{}) // init flags.passwordChanged to avoid nil pointer
	_, _, _ = flags.EndpointAndPlaintextSetting()     // allow unencrypted http communication

	testIO, _, _, _ := roxctlio.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	cmd := &scannerDownloadDBCommand{
		env: env,
	}

	version, err := cmd.versionFromCentral()
	assert.NoError(t, err)
	assert.Equal(t, "4.3.2.1", version)
}
