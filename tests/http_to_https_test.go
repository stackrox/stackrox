package tests

import (
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/require"
)

// Ensure HTTP/1.x requests to central's HTTPS endpoint result in `400`.
func TestHttpToHttps(t *testing.T) {
	url := "http://" + testutils.RoxAPIEndpoint(t)

	resp, err := http.Get(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
