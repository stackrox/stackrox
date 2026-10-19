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
	responseID             string
	assertionID            string
	userEMail              string
}

func samlResponseXML(t *testing.T, params samlResponseParams) string {
	t.Helper()
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<saml2p:Response xmlns:saml2p="urn:oasis:names:tc:SAML:2.0:protocol"
    Destination="%s"
    ID="%s"
    IssueInstant="2025-01-01T00:00:00Z"
    Version="2.0">
  <saml2:Issuer xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion">%s</saml2:Issuer>
  <saml2p:Status>
    <saml2p:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
  </saml2p:Status>
  <saml2:Assertion xmlns:saml2="urn:oasis:names:tc:SAML:2.0:assertion"
      ID="%s"
      IssueInstant="2025-01-01T00:00:00Z"
      Version="2.0">
    <saml2:Issuer>%s</saml2:Issuer>
    <saml2:Subject>
      <saml2:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">%s</saml2:NameID>
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
            xsi:type="xs:string">%s</saml2:AttributeValue>
      </saml2:Attribute>
    </saml2:AttributeStatement>
  </saml2:Assertion>
</saml2p:Response>`,
		testACSURL,
		params.responseID,
		testIdPIssuer,
		params.assertionID,
		testIdPIssuer,
		params.userEMail,
		params.subjectNotOnOrAfter.Format(time.RFC3339),
		testACSURL,
		params.conditionsNotBefore.Format(time.RFC3339),
		params.conditionsNotOnOrAfter.Format(time.RFC3339),
		params.audience,
		params.userEMail,
	)
}

type stubProvider struct {
	authproviders.Provider
	name string
}

func (s *stubProvider) Name() string { return s.name }

func newTestBackend(now time.Time, spAudience string) *backendImpl {
	audienceURI := testAudience
	if spAudience != "" {
		audienceURI = spAudience
	}
	cache := newReplayCache()
	cache.start()
	return &backendImpl{
		provider:    &stubProvider{name: "test-saml-provider"},
		replayCache: cache,
		sp: saml2.SAMLServiceProvider{
			IdentityProviderIssuer:      testIdPIssuer,
			AssertionConsumerServiceURL: testACSURL,
			ServiceProviderIssuer:       testAudience,
			AudienceURI:                 audienceURI,
			SkipSignatureValidation:     true,
			Clock:                       dsig.NewFakeClockAt(now),
		},
	}
}

// TestConsumeSAMLResponse_ReplayProtection verifies that replay attacks are prevented
// by tracking assertion IDs and rejecting duplicate assertions.
func TestConsumeSAMLResponse_ReplayProtection(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	backend := newTestBackend(now, testAudience)
	defer backend.replayCache.stop()

	xml := samlResponseXML(t, samlResponseParams{
		conditionsNotBefore:    now.Add(-5 * time.Minute),
		conditionsNotOnOrAfter: now.Add(5 * time.Minute),
		subjectNotOnOrAfter:    now.Add(10 * time.Minute),
		audience:               testAudience,
		responseID:             "_response_1_",
		assertionID:            "_assertion_1_",
		userEMail:              "attacker@evil.com",
	})
	encoded := base64.StdEncoding.EncodeToString([]byte(xml))

	// First consumption should succeed
	resp, err := backend.consumeSAMLResponse(encoded)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "attacker@evil.com", resp.Claims.UserID)

	// Second consumption of the same response should be rejected
	resp, err = backend.consumeSAMLResponse(encoded)
	require.Error(t, err)
	assert.ErrorIs(t, err, errAssertionReplay)
	assert.ErrorIs(t, err, errox.NotAuthorized)
	assert.Nil(t, resp)

	// Third attempt should also be rejected
	resp, err = backend.consumeSAMLResponse(encoded)
	require.Error(t, err)
	assert.ErrorIs(t, err, errAssertionReplay)
	assert.Nil(t, resp)
}

// TestConsumeSAMLResponse_DifferentAssertionsAccepted verifies that different
// assertions are accepted even after one has been consumed.
func TestConsumeSAMLResponse_DifferentAssertionsAccepted(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	backend := newTestBackend(now, testAudience)

	defer backend.replayCache.stop()

	// First SAML response (with assertion ID "_assertion_1")
	xml1 := samlResponseXML(t, samlResponseParams{
		conditionsNotBefore:    now.Add(-5 * time.Minute),
		conditionsNotOnOrAfter: now.Add(5 * time.Minute),
		subjectNotOnOrAfter:    now.Add(10 * time.Minute),
		audience:               testAudience,
		responseID:             "_response_1_",
		assertionID:            "_assertion_1_",
		userEMail:              "attacker@evil.com",
	})
	encoded1 := base64.StdEncoding.EncodeToString([]byte(xml1))

	// Create a second SAML response with a different assertion ID
	xml2 := samlResponseXML(t, samlResponseParams{
		conditionsNotBefore:    now.Add(-5 * time.Minute),
		conditionsNotOnOrAfter: now.Add(5 * time.Minute),
		subjectNotOnOrAfter:    now.Add(10 * time.Minute),
		audience:               testAudience,
		responseID:             "_response_2_",
		assertionID:            "_assertion_2_",
		userEMail:              "user@example.com",
	})
	encoded2 := base64.StdEncoding.EncodeToString([]byte(xml2))

	// First assertion should be accepted
	resp1, err := backend.consumeSAMLResponse(encoded1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	assert.Equal(t, "attacker@evil.com", resp1.Claims.UserID)

	// Second assertion (different ID) should also be accepted
	resp2, err := backend.consumeSAMLResponse(encoded2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	assert.Equal(t, "user@example.com", resp2.Claims.UserID)

	// But replaying the first assertion should be rejected
	resp1Replay, err := backend.consumeSAMLResponse(encoded1)
	require.Error(t, err)
	assert.ErrorIs(t, err, errAssertionReplay)
	assert.Nil(t, resp1Replay)

	// And replaying the second assertion should also be rejected
	resp2Replay, err := backend.consumeSAMLResponse(encoded2)
	require.Error(t, err)
	assert.ErrorIs(t, err, errAssertionReplay)
	assert.Nil(t, resp2Replay)
}
