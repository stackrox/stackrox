package tests

import (
	"context"
	"os"
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
		certPEM           string
		expectedResp      central.URLHasValidCertResponse_URLResult
		additionalMessage string
	}{
		{
			url: "https://untrusted-root.badssl.com",
			certPEM: `
-----BEGIN CERTIFICATE-----
MIIEmTCCAoGgAwIBAgIJAOHVqNiqXCTsMA0GCSqGSIb3DQEBCwUAMIGBMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5j
aXNjbzEPMA0GA1UECgwGQmFkU1NMMTQwMgYDVQQDDCtCYWRTU0wgVW50cnVzdGVk
IFJvb3QgQ2VydGlmaWNhdGUgQXV0aG9yaXR5MB4XDTI0MDUxNzE3NTkzM1oXDTI2
MDUxNzE3NTkzM1owYjELMAkGA1UEBhMCVVMxEzARBgNVBAgMCkNhbGlmb3JuaWEx
FjAUBgNVBAcMDVNhbiBGcmFuY2lzY28xDzANBgNVBAoMBkJhZFNTTDEVMBMGA1UE
AwwMKi5iYWRzc2wuY29tMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA
wgTs+IzuBMKz2FDVcFjMkxjrXKhoSbAitfmVnrErLHY+bMBLYExM6rK0wA+AtrD5
csmGAvlcQV0TK39xxEu86ZQuUDemZxxhjPZBQsVG0xaHJ5906wqdEVImIXNshEx5
VeTRa+gGPUgVUq2zKNuq/27/YJVKd2s58STRMbbdTcDE/FO5bUKttXz+rvUV0jNI
5yJxx8IUemwo6jdK3+pstXK0flqiFtxpsVdE2woSq97DD0d0XEEi4Zr5G5PmrSIG
KS6xukkcDCeeo/uL90ByAKySCNmMV4RTgQXL5v5rVJhAJ4XHELtzcO9pGEEHRVV8
+WQ/PSzDqXzrkxpMhtHKhQIDAQABozIwMDAJBgNVHRMEAjAAMCMGA1UdEQQcMBqC
DCouYmFkc3NsLmNvbYIKYmFkc3NsLmNvbTANBgkqhkiG9w0BAQsFAAOCAgEArxeE
TokyoO4KWzVg2euvFfP4sITwoETMBarAunrrlFgaLZ09CBxYbSSvsarhdVjGby1e
KD2ECaOXyTaB0tgq6it2nBby+k1fu4gdWWwDpCp/F2SB6nlV/ldt2pDqhkvGdNCW
j3v+YKVlM/QnJPQbVdWXVdO6WRhzIHCUZQZ/Wd/9JgE+yLd8IF0+IEbK3W/X233v
1K3gw3HPHKLSJShQyp8TNfn33IJ6J+6UlQdWPTKNI+uCr5B3Sk17n1+B9V0KdBIE
C4lv9N/3o0YxlzZD2hqHH57tmotSA0gp4oPkPwSAKumldZUusLcbVl1xPYzV0JOY
q2yMJ9FDCI1/qia3fwdkGKDJOkdz4Pn17HFy+r3Z2SPz3yxbaQC/boxxdim4Etyo
q6suC/Ztfi7x5vWpuzF/GNEO80d+uE9kr8h+qV+f385p+fS8jdEdGAsRpKNh9yDS
xs7YP5VCrm9TdEMN/TKG0qeqQD3cfS8j4h7IXR8+4NilfYbDZEfhn3ewOsXvTOec
dfj2yGeh+KmqIO28Cn0a4K5WCvFPjenz5HGcCKfGRY2qTcnSHCzotW4LQwFp9B8c
3KJEpt+0D7xSieIfR0nqf+si3ulzMViyEKLeZd+ZiqY0R1F8I3zsLwNmvMqfUXu2
7/yisXexTInYKqRh75G4BJzh8waJZvTShjjSsv4=
-----END CERTIFICATE-----`,
			// This should succeed because, even though it's a bad cert, we have configured Central to trust its root
			// on startup.
			expectedResp:      central.URLHasValidCertResponse_REQUEST_SUCCEEDED,
			additionalMessage: "This failure likely means that setting up trusted CAs with Central is broken. Look at the TRUSTED_CA_FILE being exported in the deploy scripts",
		},
		{
			url: "https://self-signed.badssl.com",
			certPEM: `-----BEGIN CERTIFICATE-----
MIIDeTCCAmGgAwIBAgIJANuSS2L+9oTlMA0GCSqGSIb3DQEBCwUAMGIxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNp
c2NvMQ8wDQYDVQQKDAZCYWRTU0wxFTATBgNVBAMMDCouYmFkc3NsLmNvbTAeFw0y
NDA1MTcxNzU5MzNaFw0yNjA1MTcxNzU5MzNaMGIxCzAJBgNVBAYTAlVTMRMwEQYD
VQQIDApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNpc2NvMQ8wDQYDVQQK
DAZCYWRTU0wxFTATBgNVBAMMDCouYmFkc3NsLmNvbTCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAMIE7PiM7gTCs9hQ1XBYzJMY61yoaEmwIrX5lZ6xKyx2
PmzAS2BMTOqytMAPgLaw+XLJhgL5XEFdEyt/ccRLvOmULlA3pmccYYz2QULFRtMW
hyefdOsKnRFSJiFzbIRMeVXk0WvoBj1IFVKtsyjbqv9u/2CVSndrOfEk0TG23U3A
xPxTuW1CrbV8/q71FdIzSOciccfCFHpsKOo3St/qbLVytH5aohbcabFXRNsKEqve
ww9HdFxBIuGa+RuT5q0iBikusbpJHAwnnqP7i/dAcgCskgjZjFeEU4EFy+b+a1SY
QCeFxxC7c3DvaRhBB0VVfPlkPz0sw6l865MaTIbRyoUCAwEAAaMyMDAwCQYDVR0T
BAIwADAjBgNVHREEHDAaggwqLmJhZHNzbC5jb22CCmJhZHNzbC5jb20wDQYJKoZI
hvcNAQELBQADggEBAH1tiJTqI9nW4Vr3q6joNV7+hNKS2OtgqBxQhMVWWWr4mRDf
ayfr4eAJkiHv8/Fvb6WqbGmzClCVNVOrfTzHeLsfROLLmlkYqXSST76XryQR6hyt
4qWqGd4M+MUNf7ty3zcVF0Yt2vqHzp4y8m+mE5nSqRarAGvDNJv+I6e4Edw19u1j
ddjiqyutdMsJkgvfNvSLQA8u7SAVjnhnoC6n2jm2wdFbrB+9rnrGje+Q8r1ERFyj
SG26SdQCiaG5QBCuDhrtLSR1N90URYCY0H6Z57sWcTKEusb95Pz6cBTLGuiNDKJq
juBzebaanR+LTh++Bleb9I0HxFFCTwlQhxo/bfY=
-----END CERTIFICATE-----`,
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
			if c.url == "https://untrusted-root.badssl.com" && os.Getenv("ORCHESTRATOR_FLAVOR") == "openshift" {
				t.Skip("temporarily skipped on OCP. TODO(ROX-24688)")
			}
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
