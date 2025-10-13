//go:build test_e2e

package tests

import (
	"net/http"
	"testing"

	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/require"
)

// Ensure HTTP/1.x requests to central's HTTPS endpoint result in `400`.
func TestHttpToHttps(t *testing.T) {
	url := "http://" + centralgrpc.RoxAPIEndpoint(t)

	resp, err := http.Get(url)
	defer testutils.SafeClientClose(resp)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
