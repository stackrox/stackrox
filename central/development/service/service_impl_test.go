package service

import (
	"context"
	_ "embed"
	"net/http"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/grpc/authn/basic"
	"github.com/stackrox/rox/pkg/uuid"
	bad_ca "github.com/stackrox/rox/tests/bad-ca"
	"github.com/stretchr/testify/suite"
)

var (
	targetEndPointNames = []string{
		"/central.DevelopmentService/ReplicateImage",
		"/central.DevelopmentService/URLHasValidCert",
		"/central.DevelopmentService/RandomData",
		"/central.DevelopmentService/EnvVars",
		"/central.DevelopmentService/ReconciliationStatsByCluster",
	}
)

func TestDevelopmentServiceAccessControl(t *testing.T) {
	suite.Run(t, new(developmentServiceAccessControlTestSuite))
}

type developmentServiceAccessControlTestSuite struct {
	suite.Suite

	svc *serviceImpl

	authProvider authproviders.Provider

	withAdminRoleCtx context.Context
	withNoneRoleCtx  context.Context
	withNoAccessCtx  context.Context
	withNoRoleCtx    context.Context
	anonymousCtx     context.Context
}

func (s *developmentServiceAccessControlTestSuite) SetupSuite() {
	s.svc = &serviceImpl{
		client: http.DefaultClient,
	}

	authProvider, err := authproviders.NewProvider(
		authproviders.WithEnabled(true),
		authproviders.WithID(uuid.NewDummy().String()),
		authproviders.WithName("Test Auth Provider"),
	)
	s.Require().NoError(err)
	s.authProvider = authProvider
	s.withAdminRoleCtx = basic.ContextWithAdminIdentity(s.T(), s.authProvider)
	s.withNoneRoleCtx = basic.ContextWithNoneIdentity(s.T(), s.authProvider)
	s.withNoAccessCtx = basic.ContextWithNoAccessIdentity(s.T(), s.authProvider)
	s.withNoRoleCtx = basic.ContextWithNoRoleIdentity(s.T(), s.authProvider)
	s.anonymousCtx = context.Background()
}

type testCase struct {
	name string
	ctx  context.Context

	expectedAuthorizerError    error
	expectedRandomServiceError error
}

func (s *developmentServiceAccessControlTestSuite) getTestCases() []testCase {
	return []testCase{
		{
			name: accesscontrol.Admin,
			ctx:  s.withAdminRoleCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    nil,
		},
		{
			name: accesscontrol.None,
			ctx:  s.withNoneRoleCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NotAuthorized,
		},
		{
			name: "No Access",
			ctx:  s.withNoAccessCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NotAuthorized,
		},
		{
			name: "No Role",
			ctx:  s.withNoRoleCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NoCredentials,
		},
		{
			name: "Anonymous",
			ctx:  s.anonymousCtx,

			expectedRandomServiceError: nil,
			expectedAuthorizerError:    errox.NoCredentials,
		},
	}
}

func (s *developmentServiceAccessControlTestSuite) TestDevelopmentServiceAuthorizer() {
	for _, endPoint := range targetEndPointNames {
		s.Run(endPoint, func() {
			for _, c := range s.getTestCases() {
				s.Run(c.name, func() {
					ctx, err := s.svc.AuthFuncOverride(c.ctx, endPoint)
					s.ErrorIs(err, c.expectedAuthorizerError)
					s.Equal(c.ctx, ctx)
				})
			}
		})
	}
}

func (s *developmentServiceAccessControlTestSuite) TestDevelopmentServiceRandomBytes() {
	const dataSize = 16
	request := &central.RandomDataRequest{}
	request.SetSize(dataSize)
	for _, c := range s.getTestCases() {
		s.Run(c.name, func() {
			rsp, err := s.svc.RandomData(c.ctx, request)
			s.ErrorIs(err, c.expectedRandomServiceError)
			s.NotNil(rsp)
			if rsp != nil {
				s.Len(rsp.GetData(), dataSize)
			}
		})
	}
}

func (s *developmentServiceAccessControlTestSuite) TestCertService() {
	cases := []struct {
		url               string
		certPEM           string
		expectedResp      central.URLHasValidCertResponse_URLResult
		additionalMessage string
	}{
		{
			url:          "https://untrusted-root.invalid",
			certPEM:      bad_ca.CustomCertPem,
			expectedResp: central.URLHasValidCertResponse_CERT_SIGNED_BY_UNKNOWN_AUTHORITY,
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

	ctx := context.Background()

	for _, c := range cases {
		s.Run(c.url, func() {
			urlhvcr := &central.URLHasValidCertRequest{}
			urlhvcr.SetUrl(c.url)
			urlhvcr.SetCertPEM(c.certPEM)
			resp, err := s.svc.URLHasValidCert(ctx, urlhvcr)
			s.NoError(err)
			s.Equal(c.expectedResp.String(), resp.GetResult().String())
		})
	}
}
