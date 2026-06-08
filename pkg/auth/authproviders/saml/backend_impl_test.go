package saml

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	saml2 "github.com/russellhaering/gosaml2"
	dsig "github.com/russellhaering/goxmldsig"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumeSAMLResponse_RejectsExpiredAssertion(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	cases := map[string]struct {
		conditionsNotBefore    time.Time
		conditionsNotOnOrAfter time.Time
		subjectNotOnOrAfter    time.Time
		wantErr                bool
	}{
		"valid assertion is accepted": {
			conditionsNotBefore:    now.Add(-5 * time.Minute),
			conditionsNotOnOrAfter: now.Add(5 * time.Minute),
			subjectNotOnOrAfter:    now.Add(10 * time.Minute),
		},
		"expired assertion is rejected": {
			conditionsNotBefore:    now.Add(-1 * time.Hour),
			conditionsNotOnOrAfter: now.Add(-30 * time.Minute),
			subjectNotOnOrAfter:    now.Add(1 * time.Hour),
			wantErr:                true,
		},
		"not-yet-valid assertion is rejected": {
			conditionsNotBefore:    now.Add(30 * time.Minute),
			conditionsNotOnOrAfter: now.Add(1 * time.Hour),
			subjectNotOnOrAfter:    now.Add(2 * time.Hour),
			wantErr:                true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			backend := newTestBackend(now)
			xml := samlResponseXML(t, samlResponseParams{
				conditionsNotBefore:    tc.conditionsNotBefore,
				conditionsNotOnOrAfter: tc.conditionsNotOnOrAfter,
				subjectNotOnOrAfter:    tc.subjectNotOnOrAfter,
				audience:               testAudience,
			})
			encoded := base64.StdEncoding.EncodeToString([]byte(xml))

			resp, err := backend.consumeSAMLResponse(encoded)

			if !tc.wantErr {
				require.NoError(t, err)
				require.NotNil(t, resp)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, errAssertionExpired)
				assert.ErrorIs(t, err, errox.NotAuthorized)
				assert.Nil(t, resp)
			}
		})
	}
}

const (
	testACSURL    = "https://stackrox.example.com/sso/providers/saml/acs"
	testIdPIssuer = "https://idp.example.com"
	testAudience  = "https://stackrox.example.com"
)

type samlResponseParams struct {
	conditionsNotBefore    time.Time
	conditionsNotOnOrAfter time.Time
	subjectNotOnOrAfter    time.Time
	audience               string
}

func samlResponseXML(t *testing.T, params samlResponseParams) string {
	t.Helper()
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<saml2p:Response xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol"
    Destination="%s"
    ID="_response_1"
    IssueInstant="2025-01-01T00:00:00Z"
    Version="2.0">
  <saml2:Issuer xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">%s</saml2:Issuer>
  <saml2p:Status>
    <saml2p:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </saml2p:Status>
  <saml2:Assertion xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion"
      ID="_assertion_1"
      IssueInstant="2025-01-01T00:00:00Z"
      Version="2.0">
    <saml2:Issuer>%s</saml2:Issuer>
    <saml2:Subject>
      <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">attacker@evil.com</saml2:NameID>
      <saml2:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
        <saml2:SubjectConfirmationData
            NotOnOrAfter="%s"
            Recipient="%s"/>
      </saml2:SubjectConfirmation>
    </saml2:Subject>
    <saml2:Conditions NotBefore="%s" NotOnOrAfter="%s">
      <saml2:AudienceRestriction>
        <saml2:Audience>%s</saml2:Audience>
      </saml2:AudienceRestriction>
    </saml2:Conditions>
    <saml2:AuthnStatement AuthnInstant="2025-01-01T00:00:00Z">
      <saml2:AuthnContext>
        <saml2:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:Password</saml2:AuthnContextClassRef>
      </saml2:AuthnContext>
    </saml2:AuthnStatement>
    <saml2:AttributeStatement>
      <saml2:Attribute Name="email">
        <saml2:AttributeValue xmlns:xs="http://www.w3.org/2001/XMLSchema"
            xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
            xsi:type="xs:string">attacker@evil.com</saml2:AttributeValue>
      </saml2:Attribute>
    </saml2:AttributeStatement>
  </saml2:Assertion>
</saml2p:Response>`,
		testACSURL,
		testIdPIssuer,
		testIdPIssuer,
		params.subjectNotOnOrAfter.Format(time.RFC3339),
		testACSURL,
		params.conditionsNotBefore.Format(time.RFC3339),
		params.conditionsNotOnOrAfter.Format(time.RFC3339),
		params.audience,
	)
}

type stubProvider struct {
	authproviders.Provider
	name string
}

func (s *stubProvider) Name() string { return s.name }

func newTestBackend(now time.Time) *backendImpl {
	return &backendImpl{
		provider: &stubProvider{name: "test-saml-provider"},
		sp: saml2.SAMLServiceProvider{
			IdentityProviderIssuer:      testIdPIssuer,
			AssertionConsumerServiceURL: testACSURL,
			SkipSignatureValidation:     true,
			Clock:                       dsig.NewFakeClockAt(now),
		},
	}
}
