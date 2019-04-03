package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCASetup(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	service := central.NewDevelopmentServiceClient(conn)

	cases := []struct {
		url               string
		expectedResp      central.URLHasValidCertResponse_URLResult
		additionalMessage string
	}{
		{
			url: "https://superfish.badssl.com",
			// This should succeed because, even though it's a bad cert, we have configured Central to trust it
			// on startup.
			expectedResp:      central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
			additionalMessage: "This failure likely means that setting up trusted CAs with Central is broken. Look at the TRUSTED_CA_FILE being exported in the deploy scripts",
		},
		{
			url:          "https://untrusted-root.badssl.com",
			expectedResp: central.URLHasValidCertResponse_CERT_SIGNED_BY_UNKNOWN_AUTHORITY,
		},
		{
			url:          "https://expired.badssl.com",
			expectedResp: central.URLHasValidCertResponse_CERT_SIGNING_AUTHORITY_VALID_BUT_OTHER_ERROR,
		},
		{
			url:          "https://google.com",
			expectedResp: central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
		},
		{
			url:          "https://doesnotexist123.com",
			expectedResp: central.URLHasValidCertResponse_OTHER_GET_ERROR,
		},
	}

	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			resp, err := service.URLHasValidCert(ctx, &central.URLHasValidCertRequest{Url: c.url})
			require.NoError(t, err)
			assert.Equal(t, c.expectedResp, resp.GetResult(), "received resp: %+v. %s", resp, c.additionalMessage)
		})
	}
}
