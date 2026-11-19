package servicecerttoken

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	commonServiceCertToken "github.com/stackrox/rox/pkg/grpc/common/authn/servicecerttoken"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	testLeeway = time.Minute

	testSensorID = "test-sensor"
)

var (
	errDummy = errors.New("dummy test error")

	testSubject = mtls.NewSubject(testSensorID, storage.ServiceType_SENSOR_SERVICE)
)

// fakeCertChainValidator is a test stand-in for authn.ValidateCertChain.
type fakeCertChainValidator struct {
	err   error
	chain []mtls.CertInfo
}

func (v *fakeCertChainValidator) ValidateClientCertificate(_ context.Context, chain []mtls.CertInfo) error {
	v.chain = chain
	return v.err
}

func newTestCA(t *testing.T) mtls.CA {
	certPEM, _, keyPEM, err := initca.New(&csr.CertificateRequest{
		CN: "Test Service CA",
	})
	require.NoError(t, err)
	ca, err := mtls.LoadCAForSigning(certPEM, keyPEM)
	require.NoError(t, err)
	return ca
}

func issueTestCert(t *testing.T, ca mtls.CA) *tls.Certificate {
	issued, err := ca.IssueCertForSubject(testSubject)
	require.NoError(t, err)
	cert, err := tls.X509KeyPair(issued.CertPEM, issued.KeyPEM)
	require.NoError(t, err)
	return &cert
}

func newTestExtractor(ca mtls.CA, validator authn.ValidateCertChain) extractor {
	return extractor{
		verifyOpts: x509.VerifyOptions{Roots: ca.CertPool()},
		maxLeeway:  testLeeway,
		validator:  validator,
	}
}

func makeRequestInfoWithToken(token string) requestinfo.RequestInfo {
	return requestinfo.RequestInfo{
		Metadata: metadata.MD{"authorization": []string{commonServiceCertToken.TokenType + " " + token}},
	}
}

func TestExtractorIdentityForRequest(t *testing.T) {
	ca := newTestCA(t)
	trustedCert := issueTestCert(t, ca)

	// A CA the extractor does not trust, and a certificate issued by it.
	untrustedCert := issueTestCert(t, newTestCA(t))

	validToken, err := commonServiceCertToken.CreateToken(trustedCert, time.Now())
	require.NoError(t, err)

	// Requests that yield neither identity nor error.
	for name, tc := range map[string]struct {
		requestInfo requestinfo.RequestInfo
	}{
		"NoIdentity-NoError - no authorization metadata": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{},
			},
		},
		"NoIdentity-NoError - non-ServiceCert authorization metadata": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{"Bearer dummyTokenData"}},
			},
		},
		"NoIdentity-NoError - bare ServiceCert prefix": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{"authorization": []string{commonServiceCertToken.TokenType}},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			extractor := newTestExtractor(ca, nil)

			id, err := extractor.IdentityForRequest(it.Context(), tc.requestInfo)
			assert.Nil(it, err)
			assert.Nil(it, id)
		})
	}

	// Error paths.
	for name, tc := range map[string]struct {
		request requestinfo.RequestInfo
		errMsg  string
	}{
		"Error: malformed token": {
			request: makeRequestInfoWithToken("not-a-token"),
			errMsg:  "could not parse service cert token",
		},
		"Error: token timestamp outside of allowed leeway": {
			request: func() requestinfo.RequestInfo {
				token, err := commonServiceCertToken.CreateToken(trustedCert, time.Now().Add(-10*testLeeway))
				require.NoError(t, err)
				return makeRequestInfoWithToken(token)
			}(),
			errMsg: "could not parse service cert token",
		},
		"Error: certificate signed by an untrusted CA": {
			request: func() requestinfo.RequestInfo {
				token, err := commonServiceCertToken.CreateToken(untrustedCert, time.Now())
				require.NoError(t, err)
				return makeRequestInfoWithToken(token)
			}(),
			errMsg: "could not verify certificate",
		},
	} {
		t.Run(name, func(it *testing.T) {
			extractor := newTestExtractor(ca, nil)

			id, err := extractor.IdentityForRequest(it.Context(), tc.request)
			assert.ErrorContains(it, err, tc.errMsg)
			assert.Nil(it, id)
		})
	}

	t.Run("Error: client certificate validation fails", func(it *testing.T) {
		validator := &fakeCertChainValidator{err: errDummy}
		extractor := newTestExtractor(ca, validator)

		id, err := extractor.IdentityForRequest(it.Context(), makeRequestInfoWithToken(validToken))
		assert.ErrorContains(it, err, "could not validate client certificate from service cert token")
		assert.Nil(it, id)
		require.NotNil(it, validator.chain)
		assert.NotEmpty(it, validator.chain)
	})

	// Success paths.
	validateServiceIdentity := func(it *testing.T, id authn.Identity) {
		require.NotNil(it, id)
		assert.Equal(it, fmt.Sprintf("mtls:%s@%v", testSensorID, trustedCert.Leaf.SerialNumber), id.UID())
		assert.Equal(it, "SENSOR_SERVICE: "+testSensorID, id.FriendlyName())
		assert.Equal(it, "SENSOR_SERVICE: "+testSensorID, id.FullName())
		protoassert.Equal(it, &storage.ServiceIdentity{
			SerialStr: trustedCert.Leaf.SerialNumber.String(),
			Type:      storage.ServiceType_SENSOR_SERVICE,
			Id:        testSensorID,
		}, id.Service())
		validFrom, validUntil := id.ValidityPeriod()
		assert.Equal(it, trustedCert.Leaf.NotBefore, validFrom)
		assert.Equal(it, trustedCert.Leaf.NotAfter, validUntil)
	}

	t.Run("Valid service cert token without validator", func(it *testing.T) {
		extractor := newTestExtractor(ca, nil)

		id, err := extractor.IdentityForRequest(it.Context(), makeRequestInfoWithToken(validToken))
		assert.Nil(it, err)
		validateServiceIdentity(it, id)
	})

	t.Run("Valid service cert token with validator", func(it *testing.T) {
		validator := &fakeCertChainValidator{}
		extractor := newTestExtractor(ca, validator)

		id, err := extractor.IdentityForRequest(it.Context(), makeRequestInfoWithToken(validToken))
		assert.Nil(it, err)
		validateServiceIdentity(it, id)
		require.NotNil(it, validator.chain)
		assert.Equal(it, trustedCert.Leaf.SerialNumber, validator.chain[0].SerialNumber)
	})
}
