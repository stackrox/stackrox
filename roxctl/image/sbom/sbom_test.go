package sbom

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stackrox/rox/roxctl/common/environment"
	"github.com/stackrox/rox/roxctl/common/flags"
	roxctlio "github.com/stackrox/rox/roxctl/common/io"
	"github.com/stackrox/rox/roxctl/common/printer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	fakeSBOM   = `{"fake":"sbom"}`
	fakeImgRef = "image.invalid/noexist:latest"
)

func TestConstruct(t *testing.T) {
	server := createServer(t, false, false)
	defer server.Close()

	env, _ := createEnv(t)

	t.Run("success", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		err := cmd.construct(c)
		assert.NoError(t, err)
		assert.Contains(t, string(cmd.requestBody), fakeImgRef)
	})

	t.Run("error on empty image", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		cmd.image = ""
		err := cmd.construct(c)
		assert.ErrorContains(t, err, "invalid reference")
	})

	t.Run("error on invalid image", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		cmd.image = ":@"
		err := cmd.construct(c)
		assert.ErrorContains(t, err, "invalid reference")
	})

	t.Run("error on invalid image digest algorithm - sha1", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		cmd.image = "registry.invalid/repo@sha1:0"
		err := cmd.construct(c)
		assert.ErrorContains(t, err, "invalid reference format")
	})

	t.Run("error on invalid image digest algorithm - sha257", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		cmd.image = "registry.invalid/repo@sha257:00000000000000000000000000000000000000000000000000000000000000000"
		err := cmd.construct(c)
		assert.ErrorContains(t, err, "unsupported digest algorithm")
	})

	t.Run("no error on valid image digest algorithm - sha256", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		cmd.image = "registry.invalid/repo@sha256:0000000000000000000000000000000000000000000000000000000000000000"
		err := cmd.construct(c)
		assert.NoError(t, err)
	})

	t.Run("no error on valid image digest algorithm - sha512", func(t *testing.T) {
		cmd, c := buildCmds(t, env)
		cmd.image = "registry.invalid/repo@sha512:00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
		err := cmd.construct(c)
		assert.NoError(t, err)
	})

	t.Run("error on invalid api endpoint", func(t *testing.T) {
		t.Setenv("ROX_ENDPOINT", "fake.invalid") // missing port breaks http client
		cmd, c := buildCmds(t, env)
		err := cmd.construct(c)
		assert.ErrorContains(t, err, "HTTP client")
	})
}

func TestGenerateSBOM(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		server := createServer(t, false, false)
		defer server.Close()

		env, out := createEnv(t)
		cmd, c := buildCmds(t, env)
		require.NoError(t, cmd.construct(c))

		err := cmd.GenerateSBOM()
		require.NoError(t, err)

		// Output is identical to what server returned.
		assert.Equal(t, out.String(), fakeSBOM)
	})

	t.Run("http error from server handled", func(t *testing.T) {
		server := createServer(t, true, false)
		defer server.Close()

		env, _ := createEnv(t)
		cmd, c := buildCmds(t, env)
		require.NoError(t, cmd.construct(c))

		err := cmd.GenerateSBOM()
		require.ErrorContains(t, err, "Error From Request Body")
	})

	t.Run("error on text/html response", func(t *testing.T) {
		server := createServer(t, false, true)
		defer server.Close()

		env, _ := createEnv(t)
		cmd, c := buildCmds(t, env)
		require.NoError(t, cmd.construct(c))

		err := cmd.GenerateSBOM()
		require.ErrorContains(t, err, "unexpected Content-Type")
	})
}

// createServer sets up a test HTTP server.
func createServer(t *testing.T, retErr bool, htmlResponse bool) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		if retErr {
			rw.WriteHeader(http.StatusInternalServerError)
			_, _ = rw.Write([]byte(`{"code":13,"message":"Error From Request Body"}`))
			return
		}

		if htmlResponse {
			rw.Header().Add("Content-Type", "text/html; charset=utf-8")
			_, _ = rw.Write([]byte("<html><body>Hello</body></html>"))
			return

		}

		_, _ = rw.Write([]byte(fakeSBOM))
	}))

	t.Setenv("ROX_ENDPOINT", server.URL)
	t.Setenv("ROX_API_TOKEN", "fake")

	return server
}

// Creates and returns a test CLI environment and stdout buffer.
func createEnv(t *testing.T) (environment.Environment, *bytes.Buffer) {
	testIO, _, out, _ := roxctlio.TestIO()
	env := environment.NewTestCLIEnvironment(t, testIO, printer.DefaultColorPrinter())

	return env, out
}

// buildCmds builds the SBOM and Cobra command objects, to be used throughout the
// various tests.
func buildCmds(t *testing.T, env environment.Environment) (*imageSBOMCommand, *cobra.Command) {
	cmd := &imageSBOMCommand{env: env, image: fakeImgRef}

	cobraCmd := &cobra.Command{}
	flags.AddTimeoutWithDefault(cobraCmd, 10*time.Minute)
	flags.AddCentralAuthFlags(cobraCmd)

	t.Setenv("ROX_INSECURE_CLIENT", "true")
	t.Setenv("ROX_CLIENT_MAX_RETRIES", "0")

	return cmd, cobraCmd
}
