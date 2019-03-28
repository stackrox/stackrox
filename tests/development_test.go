package tests

import (
	"context"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/testutils"
	"github.com/stretchr/testify/assert"
)

func TestDevelopmentServer(t *testing.T) {
	conn := testutils.GRPCConnectionToCentral(t)

	service := central.NewDevelopmentServiceClient(conn)

	cases := []struct {
		url          string
		expectedResp central.URLHasValidCertResponse_URLResult
	}{
		{
			url:          "https://superfish.badssl.com",
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
			a := assert.New(t)

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			resp, err := service.URLHasValidCert(ctx, &central.URLHasValidCertRequest{Url: c.url})
			a.NoError(err)
			a.Equal(c.expectedResp, resp.Result, "received resp: %+v", resp)
		})
	}
}
