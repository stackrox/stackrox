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
			roundTripTypedServiceCertificateSet, _, err := ConvertFileMapToTypedServiceCertificateSet(fileMap)
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

func TestConvertFileMapToTypedServiceCertificateSet(t *testing.T) {
	cases := []struct {
		description                        string
		input                              map[string]string
		expectedErr                        bool
		expectedUnknownServices            []string
		expectedTypedServiceCertificateSet *storage.TypedServiceCertificateSet
	}{
		{
			description:                        "empty",
			input:                              map[string]string{},
			expectedErr:                        false,
			expectedUnknownServices:            nil,
			expectedTypedServiceCertificateSet: nil,
		},
		{
			description: "known services",
			input: map[string]string{
				"admission-control-cert.pem": "cert 1",
				"admission-control-key.pem":  "key 1",
			},
			expectedErr:             false,
			expectedUnknownServices: nil,
			expectedTypedServiceCertificateSet: &storage.TypedServiceCertificateSet{
				CaPem: nil,
				ServiceCerts: []*storage.TypedServiceCertificate{
					{
						ServiceType: storage.ServiceType_ADMISSION_CONTROL_SERVICE,
						Cert: &storage.ServiceCertificate{
							CertPem: []byte("cert 1"),
							KeyPem:  []byte("key 1"),
						},
					},
				},
			},
		},
		{
			description: "mixed known and unknown",
			input: map[string]string{
				"admission-control-cert.pem": "cert 1",
				"admission-control-key.pem":  "key 1",
				"foo-cert.pem":               "cert 2",
				"foo-key.pem":                "key 2",
				"bar-cert.pem":               "cert 3",
				"bar-key.pem":                "key 3",
			},
			expectedErr:             false,
			expectedUnknownServices: []string{"foo", "bar"},
			expectedTypedServiceCertificateSet: &storage.TypedServiceCertificateSet{
				CaPem: nil,
				ServiceCerts: []*storage.TypedServiceCertificate{
					{
						ServiceType: storage.ServiceType_ADMISSION_CONTROL_SERVICE,
						Cert: &storage.ServiceCertificate{
							CertPem: []byte("cert 1"),
							KeyPem:  []byte("key 1"),
						},
					},
				},
			},
		},
		{
			description: "invalid file names",
			input: map[string]string{
				"admission-control-cert.pem": "cert 1",
				"admission-control.pem":      "bogus file name",
			},
			expectedErr:             true,
			expectedUnknownServices: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			// Convert input FileMap into a TypedServiceCertificateSet.
			inputFileMap := c.input
			typedServiceCertificateSet, unknownServices, err := ConvertFileMapToTypedServiceCertificateSet(inputFileMap)
			if c.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, typedServiceCertificateSet)
				assert.Nil(t, unknownServices)
				return
			} else {
				assert.NoError(t, err)
			}
			expectedUnknownServices := c.expectedUnknownServices
			if expectedUnknownServices != nil {
				slices.Sort(expectedUnknownServices)
			}

			assert.Equal(t, c.expectedUnknownServices, unknownServices)
			expectedTypedServiceCertificates := c.expectedTypedServiceCertificateSet.GetServiceCerts()
			typedServiceCertificates := typedServiceCertificateSet.GetServiceCerts()

			assert.Equal(t, c.expectedTypedServiceCertificateSet.GetCaPem(), typedServiceCertificateSet.GetCaPem())

			if !protoutils.SlicesEqual(expectedTypedServiceCertificates, typedServiceCertificates) {
				assert.Fail(t,
					fmt.Sprintf("Not equal for test case %q:\n"+
						"expected typed service certs: %v\n"+
						"typed service certs: %v\n", c.description, expectedTypedServiceCertificates, typedServiceCertificates))
			}

			if len(c.expectedUnknownServices) == 0 {
				// Convert TypedServiceCertificateSet back to FileMap.
				roundTripFileMap, err := ConvertTypedServiceCertificateSetToFileMap(typedServiceCertificateSet)
				assert.NoError(t, err)
				assert.Equal(t, inputFileMap, roundTripFileMap)
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
