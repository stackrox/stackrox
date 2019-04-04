package tlscheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func handler(w http.ResponseWriter, r *http.Request) {}

func TestTLS(t *testing.T) {
	httpServer := httptest.NewServer(http.HandlerFunc(handler))
	defer httpServer.Close()

	tls, err := CheckTLS(httpServer.Listener.Addr().String())
	require.NoError(t, err)
	assert.False(t, tls)

	httpsServer := httptest.NewTLSServer(http.HandlerFunc(handler))
	tls, err = CheckTLS(httpsServer.Listener.Addr().String())
	require.NoError(t, err)
	assert.False(t, tls)
}
