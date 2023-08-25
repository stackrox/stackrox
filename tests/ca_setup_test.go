package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func centralIsReleaseBuild(conn *grpc.ClientConn, t *testing.T) bool {
	client := v1.NewMetadataServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	metadata, err := client.GetMetadata(ctx, &v1.Empty{})
	require.NoError(t, err)
	return metadata.ReleaseBuild
}

func TestCASetup(t *testing.T) {
	t.Parallel()

	conn := centralgrpc.GRPCConnectionToCentral(t)
	service := central.NewDevelopmentServiceClient(conn)

	isReleaseBuild := centralIsReleaseBuild(conn, t)
	// Can't run these tests on a release build. But also let's assert
	// that the development service is not available.
	if isReleaseBuild {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		resp, err := service.URLHasValidCert(ctx, &central.URLHasValidCertRequest{})
		require.Nil(t, resp)
		require.Error(t, err)
		require.Equal(t, codes.Unimplemented, status.Code(err))
		return
	}

	cases := []struct {
		url               string
		expectedResp      central.URLHasValidCertResponse_URLResult
		additionalMessage string
	}{
		{
			url: "https://untrusted-root.badssl.com",
			// This should succeed because, even though it's a bad cert, we have configured Central to trust its root
			// on startup.
			expectedResp:      central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
			additionalMessage: "This failure likely means that setting up trusted CAs with Central is broken. Look at the TRUSTED_CA_FILE being exported in the deploy scripts",
		},
		{
			url:          "https://self-signed.badssl.com",
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
			internalServiceTimeout := 20 * time.Second
			testTimeoutPadding := 500 * time.Millisecond
			ctx, cancel := context.WithTimeout(context.Background(), internalServiceTimeout+testTimeoutPadding)
			defer cancel()
			resp, err := service.URLHasValidCert(ctx, &central.URLHasValidCertRequest{Url: c.url})
			require.NoError(t, err)
			assert.Equal(t, c.expectedResp, resp.GetResult(), "received resp: %+v. %s", resp, c.additionalMessage)
		})
	}
}
