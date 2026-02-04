package m2m

import (
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

type authenticatedRoundTripper struct {
	roundTripper http.RoundTripper
	tokenReader  func() ([]byte, error)
}

// RoundTrip here inserts an HTTP "Authorization" header with the given bearer token.
func (a *authenticatedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// It's generally not recommended to make more than one actual round trip inside the RoundTrip function,
	// but we have no reasonable alternative if we want to utilize the oidc provider interface.
	// NOTE: This has only been tested with GET requests used by the oidc library to fetch JWKS config.

	// First try without any auth header.
	resp, err := a.roundTripper.RoundTrip(req)
	if err == nil && resp.StatusCode >= 400 {
		// If the Body is not both read to EOF and closed, the Client's
		// underlying RoundTripper (typically Transport) may not be able to
		// re-use a persistent TCP connection to the server for a subsequent
		// "keep-alive" request.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		// GKE's issuer endpoint responds with HTTP 400 if Authorization header is set.
		// At the same time, the Kube docs indicate that auth should be required by default:
		// https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#service-account-issuer-discovery
		//
		// Thus, try first with no auth.
		// If a response was received but it was a 4xx/5xx status code, try with the auth header.
		// Note that we don't try with the auth header if a proper HTTP response was not received, i.e. err != nil.
		log.Warnf("Unauthenticated oidc config request failed with %d. Trying again with a token.", resp.StatusCode)

		// Service account token is rotated hourly. We need to read it every time.
		token, err := a.tokenReader()
		if err != nil {
			return nil, errors.Wrap(err, "failed to read the token")
		}

		authReq := req.Clone(req.Context())
		authReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", string(token)))
		return a.roundTripper.RoundTrip(authReq)
	}
	return resp, err
}
