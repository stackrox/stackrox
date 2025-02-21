package httputil

import (
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultTransport(t *testing.T) {
	testServer := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "", r.Header.Get(InsecureSkipTLSVerifyHeader))
	}))
	testServer.StartTLS()
	t.Cleanup(testServer.Close)

	client := testServer.Client()
	client.Transport = NewInsecureCapableTransport(http.DefaultTransport.(*http.Transport))

	// Secure transport.
	// Because we overwrote the transport, we will not know about the server's certs.
	// Therefore, this is expected to fail.
	_, err := client.Get(testServer.URL)
	assert.ErrorAs(t, err, &x509.UnknownAuthorityError{})

	// Insecure transport.
	req, err := http.NewRequest("GET", testServer.URL, nil)
	require.NoError(t, err)
	req.Header.Set(InsecureSkipTLSVerifyHeader, "true")
	resp, err := client.Do(req)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}
