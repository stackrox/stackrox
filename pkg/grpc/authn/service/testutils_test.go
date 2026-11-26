package service

import (
	"context"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/set"
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
