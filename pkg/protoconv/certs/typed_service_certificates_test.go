package protoconv

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func makeTypedServiceCert(serviceType storage.ServiceType, certPem string, keyPem string) *storage.TypedServiceCertificate {
	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: []byte(certPem),
			KeyPem:  []byte(keyPem),
		},
	}
}

func TestConvertTypedServiceCertificateSetToFileMap(t *testing.T) {
	cases := []struct {
		description string
		input       *storage.TypedServiceCertificateSet
		expected    map[string]string
	}{
		{
			description: "empty",
			input: &storage.TypedServiceCertificateSet{
				ServiceCerts: []*storage.TypedServiceCertificate{},
			},
			expected: map[string]string{},
		},
		{
			description: "two certs",
			input: &storage.TypedServiceCertificateSet{
				CaPem: []byte("ca cert"),
				ServiceCerts: []*storage.TypedServiceCertificate{
					makeTypedServiceCert(storage.ServiceType_ADMISSION_CONTROL_SERVICE, "cert 1", "key 1"),
					makeTypedServiceCert(storage.ServiceType_SCANNER_V4_INDEXER_SERVICE, "cert 2", "key 2"),
				},
			},
			expected: map[string]string{
				"ca-cert.pem":                 "ca cert",
				"admission-control-cert.pem":  "cert 1",
				"admission-control-key.pem":   "key 1",
				"scanner-v4-indexer-cert.pem": "cert 2",
				"scanner-v4-indexer-key.pem":  "key 2",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			inputTypedServiceCertificateSet := c.input
			fileMap, err := ConvertTypedServiceCertificateSetToFileMap(inputTypedServiceCertificateSet)
			assert.NoError(t, err)
			assert.Equal(t, c.expected, fileMap)
			// roundTripTypedServiceCertificateSet := ConvertFileMapToTypedServiceCertificateSet(fileMap)
			// assert.Equal(t, inputTypedServiceCertificateSet, roundTripTypedServiceCertificateSet)
		})
	}
}
