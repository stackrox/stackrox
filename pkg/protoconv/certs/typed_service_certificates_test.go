package protoconv

import (
	"cmp"
	"fmt"
	"slices"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
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
		description     string
		input           *storage.TypedServiceCertificateSet
		expectedFileMap map[string]string
	}{
		{
			description: "empty",
			input: &storage.TypedServiceCertificateSet{
				ServiceCerts: []*storage.TypedServiceCertificate{},
			},
			expectedFileMap: map[string]string{},
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
			expectedFileMap: map[string]string{
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
			// Convert input TypedServiceCertificateSet to a FileMap.
			inputTypedServiceCertificateSet := c.input
			inputCa := inputTypedServiceCertificateSet.GetCaPem()
			inputCerts := inputTypedServiceCertificateSet.GetServiceCerts()
			fileMap, err := ConvertTypedServiceCertificateSetToFileMap(inputTypedServiceCertificateSet)
			assert.NoError(t, err)
			assert.Equal(t, c.expectedFileMap, fileMap)

			// Convert FileMap back to TypedServiceCertificateSet.
			roundTripTypedServiceCertificateSet, err := ConvertFileMapToTypedServiceCertificateSet(fileMap)
			assert.NoError(t, err)
			roundTripCa := roundTripTypedServiceCertificateSet.GetCaPem()
			roundTripCerts := roundTripTypedServiceCertificateSet.GetServiceCerts()

			// Check if the result matches the original TypedServiceCertificateSet we started with.
			assert.Equal(t, inputCa, roundTripCa, "CAs differ")
			// Note that converting from a slice to a map and back to a slice can change the order, hence the sorting.
			sortTypedServiceCertificateSlice(inputCerts)
			sortTypedServiceCertificateSlice(roundTripCerts)
			if !protoutils.SlicesEqual(inputCerts, roundTripCerts) {
				assert.Fail(t,
					fmt.Sprintf("Not equal for test case %q:\n"+
						"input certs: %v\n"+
						"round-trip certs: %v\n", c.description, inputCerts, roundTripCerts))
			}
		})
	}
}

// Sort TypedServiceCertificate slices by their service type.
func sortTypedServiceCertificateSlice(certs []*storage.TypedServiceCertificate) {
	slices.SortFunc(certs, func(a, b *storage.TypedServiceCertificate) int {
		return cmp.Compare(a.GetServiceType(), b.GetServiceType())
	})
}
