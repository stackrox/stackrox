package service

import (
	"context"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/cryptoutils"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"
)

const (
	testCAFingerprint = "test-ca-fingerprint"

	testSensorID     = "test-sensor"
	testSensorSerial = 42
)

var (
	errDummy = errors.New("dummy test error")

	testNotBefore = time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	testNotAfter  = time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC)
)

// fakeCertChainValidator is a test stand-in for authn.ValidateCertChain.
type fakeCertChainValidator struct {
	err    error
	called int
	chain  []mtls.CertInfo
}

func (v *fakeCertChainValidator) ValidateClientCertificate(_ context.Context, chain []mtls.CertInfo) error {
	v.called++
	v.chain = chain
	return v.err
}

func sensorCertInfo() mtls.CertInfo {
	return mtls.CertInfo{
		Subject:         pkix.Name{CommonName: "SENSOR_SERVICE: " + testSensorID},
		NotBefore:       testNotBefore,
		NotAfter:        testNotAfter,
		SerialNumber:    big.NewInt(testSensorSerial),
		CertFingerprint: "test-sensor-fingerprint",
	}
}

func caCertInfo(fingerprint string) mtls.CertInfo {
	return mtls.CertInfo{
		Subject:         pkix.Name{CommonName: "Test CA"},
		CertFingerprint: fingerprint,
	}
}

func newTestExtractor(validator authn.ValidateCertChain) extractor {
	fingerprints := set.NewSet[string]()
	fingerprints.Add(testCAFingerprint)
	return extractor{
		trustedCAFingerprints: fingerprints,
		validator:             validator,
	}
}

func TestExtractorIdentityForRequest(t *testing.T) {
	// Requests that yield neither identity nor error.
	for name, tc := range map[string]struct {
		requestInfo requestinfo.RequestInfo
	}{
		"NoIdentity-NoError - no verified chains": {
			requestInfo: requestinfo.RequestInfo{
				Metadata: metadata.MD{},
			},
		},
		"NoIdentity-NoError - multiple verified chains": {
			requestInfo: requestinfo.RequestInfo{
				VerifiedChains: [][]mtls.CertInfo{
					{{}, {}},
					{{}, {}},
				},
			},
		},
		"NoIdentity-NoError - chain rooted at an untrusted CA": {
			requestInfo: requestinfo.RequestInfo{
				VerifiedChains: [][]mtls.CertInfo{
					{sensorCertInfo(), caCertInfo("untrusted-ca-fingerprint")},
				},
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			extractor := newTestExtractor(nil)

			id, err := extractor.IdentityForRequest(it.Context(), tc.requestInfo)
			assert.Nil(it, err)
			assert.Nil(it, id)
		})
	}

	// Error paths.
	t.Run("Error: client certificate validation fails", func(it *testing.T) {
		validator := &fakeCertChainValidator{err: errDummy}
		extractor := newTestExtractor(validator)
		chain := []mtls.CertInfo{sensorCertInfo(), caCertInfo(testCAFingerprint)}
		requestInfo := requestinfo.RequestInfo{
			VerifiedChains: [][]mtls.CertInfo{chain},
		}

		id, err := extractor.IdentityForRequest(it.Context(), requestInfo)
		assert.ErrorContains(it, err, "client certificate validation failed")
		assert.Nil(it, id)
		assert.Equal(it, 1, validator.called)
		assert.Equal(it, chain, validator.chain)
	})

	// Success paths.
	expectedUID := fmt.Sprintf("mtls:%s@%v", testSensorID, big.NewInt(testSensorSerial))
	validateServiceIdentity := func(it *testing.T, id authn.Identity) {
		require.NotNil(it, id)
		assert.Equal(it, expectedUID, id.UID())
		assert.Equal(it, "SENSOR_SERVICE: "+testSensorID, id.FriendlyName())
		assert.Equal(it, "SENSOR_SERVICE: "+testSensorID, id.FullName())
	}

	t.Run("Valid client certificate without validator", func(it *testing.T) {
		extractor := newTestExtractor(nil)
		requestInfo := requestinfo.RequestInfo{
			VerifiedChains: [][]mtls.CertInfo{
				{sensorCertInfo(), caCertInfo(testCAFingerprint)},
			},
		}

		id, err := extractor.IdentityForRequest(it.Context(), requestInfo)
		assert.Nil(it, err)
		validateServiceIdentity(it, id)
	})

	t.Run("Valid client certificate with validator", func(it *testing.T) {
		validator := &fakeCertChainValidator{}
		extractor := newTestExtractor(validator)
		requestInfo := requestinfo.RequestInfo{
			VerifiedChains: [][]mtls.CertInfo{
				{sensorCertInfo(), caCertInfo(testCAFingerprint)},
			},
		}

		id, err := extractor.IdentityForRequest(it.Context(), requestInfo)
		assert.Nil(it, err)
		validateServiceIdentity(it, id)
		assert.Equal(it, 1, validator.called)
	})
}

func TestNewExtractor(t *testing.T) {
	t.Setenv(mtls.CAFileEnvName, "../../../../tools/local-sensor/certs/caCert.pem")

	extractor, err := NewExtractor()
	require.NoError(t, err)
	require.NotNil(t, extractor)

	ca, _, err := mtls.CACert()
	require.NoError(t, err)

	// A chain rooted at the CA picked up by NewExtractor must yield an identity.
	requestInfo := requestinfo.RequestInfo{
		VerifiedChains: [][]mtls.CertInfo{
			{sensorCertInfo(), {CertFingerprint: cryptoutils.CertFingerprint(ca)}},
		},
	}

	id, err := extractor.IdentityForRequest(t.Context(), requestInfo)
	assert.Nil(t, err)
	require.NotNil(t, id)
	assert.Equal(t, fmt.Sprintf("mtls:%s@%v", testSensorID, big.NewInt(testSensorSerial)), id.UID())
}
