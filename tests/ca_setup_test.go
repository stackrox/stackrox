//go:build test_e2e

package tests

import (
	"context"
	"testing"
	"time"

	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/testutils/centralgrpc"
	bad_ca "github.com/stackrox/rox/tests/bad-ca"
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
		certPEM           string
		expectedResp      central.URLHasValidCertResponse_URLResult
		additionalMessage string
	}{
		{
			url:     "https://untrusted-root.invalid",
			certPEM: bad_ca.CustomCertPem,
			// This should succeed because, even though it's a bad cert, we have configured Central to trust its root
			// on startup.
			expectedResp:      central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
			additionalMessage: "This failure likely means that setting up trusted CAs with Central is broken. Look at the TRUSTED_CA_FILE being exported in the deploy scripts",
		},
		{
			url:          "https://self-signed.invalid",
			certPEM:      bad_ca.SelfSignedCertPem,
			expectedResp: central.URLHasValidCertResponse_CERT_SIGNED_BY_UNKNOWN_AUTHORITY,
		},
		{
			url: "https://expired.badssl.com",
			certPEM: `
-----BEGIN CERTIFICATE-----
MIIFSzCCBDOgAwIBAgIQSueVSfqavj8QDxekeOFpCTANBgkqhkiG9w0BAQsFADCB
kDELMAkGA1UEBhMCR0IxGzAZBgNVBAgTEkdyZWF0ZXIgTWFuY2hlc3RlcjEQMA4G
A1UEBxMHU2FsZm9yZDEaMBgGA1UEChMRQ09NT0RPIENBIExpbWl0ZWQxNjA0BgNV
BAMTLUNPTU9ETyBSU0EgRG9tYWluIFZhbGlkYXRpb24gU2VjdXJlIFNlcnZlciBD
QTAeFw0xNTA0MDkwMDAwMDBaFw0xNTA0MTIyMzU5NTlaMFkxITAfBgNVBAsTGERv
bWFpbiBDb250cm9sIFZhbGlkYXRlZDEdMBsGA1UECxMUUG9zaXRpdmVTU0wgV2ls
ZGNhcmQxFTATBgNVBAMUDCouYmFkc3NsLmNvbTCCASIwDQYJKoZIhvcNAQEBBQAD
ggEPADCCAQoCggEBAMIE7PiM7gTCs9hQ1XBYzJMY61yoaEmwIrX5lZ6xKyx2PmzA
S2BMTOqytMAPgLaw+XLJhgL5XEFdEyt/ccRLvOmULlA3pmccYYz2QULFRtMWhyef
dOsKnRFSJiFzbIRMeVXk0WvoBj1IFVKtsyjbqv9u/2CVSndrOfEk0TG23U3AxPxT
uW1CrbV8/q71FdIzSOciccfCFHpsKOo3St/qbLVytH5aohbcabFXRNsKEqveww9H
dFxBIuGa+RuT5q0iBikusbpJHAwnnqP7i/dAcgCskgjZjFeEU4EFy+b+a1SYQCeF
xxC7c3DvaRhBB0VVfPlkPz0sw6l865MaTIbRyoUCAwEAAaOCAdUwggHRMB8GA1Ud
IwQYMBaAFJCvajqUWgvYkOoSVnPfQ7Q6KNrnMB0GA1UdDgQWBBSd7sF7gQs6R2lx
GH0RN5O8pRs/+zAOBgNVHQ8BAf8EBAMCBaAwDAYDVR0TAQH/BAIwADAdBgNVHSUE
FjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwTwYDVR0gBEgwRjA6BgsrBgEEAbIxAQIC
BzArMCkGCCsGAQUFBwIBFh1odHRwczovL3NlY3VyZS5jb21vZG8uY29tL0NQUzAI
BgZngQwBAgEwVAYDVR0fBE0wSzBJoEegRYZDaHR0cDovL2NybC5jb21vZG9jYS5j
b20vQ09NT0RPUlNBRG9tYWluVmFsaWRhdGlvblNlY3VyZVNlcnZlckNBLmNybDCB
hQYIKwYBBQUHAQEEeTB3ME8GCCsGAQUFBzAChkNodHRwOi8vY3J0LmNvbW9kb2Nh
LmNvbS9DT01PRE9SU0FEb21haW5WYWxpZGF0aW9uU2VjdXJlU2VydmVyQ0EuY3J0
MCQGCCsGAQUFBzABhhhodHRwOi8vb2NzcC5jb21vZG9jYS5jb20wIwYDVR0RBBww
GoIMKi5iYWRzc2wuY29tggpiYWRzc2wuY29tMA0GCSqGSIb3DQEBCwUAA4IBAQBq
evHa/wMHcnjFZqFPRkMOXxQhjHUa6zbgH6QQFezaMyV8O7UKxwE4PSf9WNnM6i1p
OXy+l+8L1gtY54x/v7NMHfO3kICmNnwUW+wHLQI+G1tjWxWrAPofOxkt3+IjEBEH
fnJ/4r+3ABuYLyw/zoWaJ4wQIghBK4o+gk783SHGVnRwpDTysUCeK1iiWQ8dSO/r
ET7BSp68ZVVtxqPv1dSWzfGuJ/ekVxQ8lEEFeouhN0fX9X3c+s5vMaKwjOrMEpsi
8TRwz311SotoKQwe6Zaoz7ASH1wq7mcvf71z81oBIgxw+s1F73hczg36TuHvzmWf
RwxPuzZEaFZcVlmtqoq8
-----END CERTIFICATE-----`,
			expectedResp: central.URLHasValidCertResponse_CERT_SIGNING_AUTHORITY_VALID_BUT_OTHER_ERROR,
		},
		{
			url:          "https://connectivitycheck.gstatic.com/",
			expectedResp: central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
		},
		{
			url:          "https://test.invalid",
			expectedResp: central.URLHasValidCertResponse_OTHER_GET_ERROR,
		},
	}

	for _, c := range cases {
		t.Run(c.url, func(t *testing.T) {
			internalServiceTimeout := 20 * time.Second
			testTimeoutPadding := 500 * time.Millisecond
			ctx, cancel := context.WithTimeout(context.Background(), internalServiceTimeout+testTimeoutPadding)
			defer cancel()
			resp, err := service.URLHasValidCert(ctx, &central.URLHasValidCertRequest{
				Url:     c.url,
				CertPEM: c.certPEM,
			})
			require.NoError(t, err)
			t.Logf("received resp: %+v. %s", resp, c.additionalMessage)
			assert.Equal(t, c.expectedResp, resp.GetResult())
		})
	}
}
