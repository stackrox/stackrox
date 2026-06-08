package saml

import (
	"encoding/base64"
	"errors"
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

func TestValidateAssertionWarnings(t *testing.T) {
	cases := map[string]struct {
		warningInfo *saml2.WarningInfo
		expectedErr error
	}{
		"nil WarningInfo should pass": {
			warningInfo: nil,
		},
		"empty WarningInfo should pass": {
			warningInfo: &saml2.WarningInfo{},
		},
		"InvalidTime should be rejected": {
			warningInfo: &saml2.WarningInfo{InvalidTime: true},
			expectedErr: errAssertionExpired,
		},
		"NotInAudience should be rejected": {
			warningInfo: &saml2.WarningInfo{NotInAudience: true},
			expectedErr: errAssertionAudienceMismatch,
		},
		"InvalidTime takes precedence over NotInAudience": {
			warningInfo: &saml2.WarningInfo{InvalidTime: true, NotInAudience: true},
			expectedErr: errAssertionExpired,
		},
		"OneTimeUse alone should pass": {
			warningInfo: &saml2.WarningInfo{OneTimeUse: true},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateAssertionWarnings(tc.warningInfo)
			if tc.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.expectedErr)
				assert.ErrorIs(t, err, errox.NotAuthorized)
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

func newTestBackend(now time.Time, spAudience string) *backendImpl {
	audienceURI := testAudience
	audienceConfigured := false
	if spAudience != "" {
		audienceURI = spAudience
		audienceConfigured = true
	}
	return &backendImpl{
		provider:           &stubProvider{name: "test-saml-provider"},
		audienceConfigured: audienceConfigured,
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

// TestConsumeSAMLResponse_AssertionValidityEnforcement demonstrates the security
// fix for SRX-C01 (SAML Assertion Validity Window & Audience Restriction).
//
// Before the fix, consumeSAMLResponse silently ignored WarningInfo from
// gosaml2, allowing:
//   - (a) Replay of expired assertions (InvalidTime not checked).
//   - (b) Cross-SP impersonation on shared IdPs (NotInAudience not checked).
func TestConsumeSAMLResponse_AssertionValidityEnforcement(t *testing.T) {
	// Fixed point in time for the fake clock.
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	cases := map[string]struct {
		spAudience             string
		conditionsNotBefore    time.Time
		conditionsNotOnOrAfter time.Time
		subjectNotOnOrAfter    time.Time
		audience               string
		wantErr                error
	}{
		"valid assertion is accepted": {
			spAudience:             testAudience,
			conditionsNotBefore:    now.Add(-5 * time.Minute),
			conditionsNotOnOrAfter: now.Add(5 * time.Minute),
			subjectNotOnOrAfter:    now.Add(10 * time.Minute),
			audience:               testAudience,
		},

		// Attack (a): Assertion replay.
		"expired assertion is rejected (replay attack)": {
			spAudience:             testAudience,
			conditionsNotBefore:    now.Add(-1 * time.Hour),
			conditionsNotOnOrAfter: now.Add(-30 * time.Minute),
			subjectNotOnOrAfter:    now.Add(1 * time.Hour),
			audience:               testAudience,
			wantErr:                errAssertionExpired,
		},

		// Attack (a) variant: assertion not yet valid.
		"not-yet-valid assertion is rejected": {
			spAudience:             testAudience,
			conditionsNotBefore:    now.Add(30 * time.Minute),
			conditionsNotOnOrAfter: now.Add(1 * time.Hour),
			subjectNotOnOrAfter:    now.Add(2 * time.Hour),
			audience:               testAudience,
			wantErr:                errAssertionExpired,
		},

		// Attack (b): Cross-SP impersonation with explicit audience configured.
		"wrong audience is rejected when sp_audience is set": {
			spAudience:             testAudience,
			conditionsNotBefore:    now.Add(-5 * time.Minute),
			conditionsNotOnOrAfter: now.Add(5 * time.Minute),
			subjectNotOnOrAfter:    now.Add(10 * time.Minute),
			audience:               "https://other-app.example.com",
			wantErr:                errAssertionAudienceMismatch,
		},

		// Backwards compat: no sp_audience, assertion matches SP issuer → accept silently.
		"no sp_audience with matching issuer accepts silently": {
			spAudience:             "",
			conditionsNotBefore:    now.Add(-5 * time.Minute),
			conditionsNotOnOrAfter: now.Add(5 * time.Minute),
			subjectNotOnOrAfter:    now.Add(10 * time.Minute),
			audience:               testAudience,
		},

		// Backwards compat: no sp_audience, assertion doesn't match SP issuer → accept with warning.
		"no sp_audience with mismatched issuer accepts with warning": {
			spAudience:             "",
			conditionsNotBefore:    now.Add(-5 * time.Minute),
			conditionsNotOnOrAfter: now.Add(5 * time.Minute),
			subjectNotOnOrAfter:    now.Add(10 * time.Minute),
			audience:               "https://different-audience.example.com",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			backend := newTestBackend(now, tc.spAudience)
			xml := samlResponseXML(t, samlResponseParams{
				conditionsNotBefore:    tc.conditionsNotBefore,
				conditionsNotOnOrAfter: tc.conditionsNotOnOrAfter,
				subjectNotOnOrAfter:    tc.subjectNotOnOrAfter,
				audience:               tc.audience,
			})
			encoded := base64.StdEncoding.EncodeToString([]byte(xml))

			resp, err := backend.consumeSAMLResponse(encoded)

			if tc.wantErr == nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, "attacker@evil.com", resp.Claims.UserID)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
				assert.ErrorIs(t, err, errox.NotAuthorized)
				assert.Nil(t, resp)
				if errors.Is(tc.wantErr, errAssertionAudienceMismatch) {
					assert.Contains(t, err.Error(), testAudience)
					assert.Contains(t, err.Error(), tc.audience)
				}
			}
		})
	}
}
