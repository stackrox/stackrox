package protoconv

import (
	"cmp"
	"embed"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoutils"
	"github.com/stackrox/rox/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

//go:embed testdata/*
var testdata embed.FS

type TypedServiceCertificateSuite struct {
	suite.Suite
}

func testFile(name string) []byte {
	path := fmt.Sprintf("testdata/%s", name)
	content, err := testdata.ReadFile(path)
	utils.Must(err)
	content = []byte(strings.ReplaceAll(string(content), "PRIVATE TEST KEY", "PRIVATE KEY"))
	return content
}

var (
	caCert        = testFile("ca-cert.pem")
	collectorCert = testFile("collector-cert.pem")
	collectorKey  = testFile("collector-key.pem")
	sensorCert    = testFile("sensor-cert.pem")
	sensorKey     = testFile("sensor-key.pem")
	bogusCert     = testFile("bogus-cert.pem")
)

func makeTypedServiceCert(serviceType storage.ServiceType, certPem []byte, keyPem []byte) *storage.TypedServiceCertificate {
	return &storage.TypedServiceCertificate{
		ServiceType: serviceType,
		Cert: &storage.ServiceCertificate{
			CertPem: certPem,
			KeyPem:  keyPem,
		},
	}
}

func TestConvertTypedServiceCertificateSetToFileMap(t *testing.T) {
	cases := []struct {
		description     string
		input           *storage.TypedServiceCertificateSet
		expectedFileMap map[string][]byte
		expectedErr     bool
	}{
		{
			description: "empty",
			input:       &storage.TypedServiceCertificateSet{},
			expectedErr: true,
		},
		{
			description: "no CA",
			input: &storage.TypedServiceCertificateSet{
				ServiceCerts: []*storage.TypedServiceCertificate{
					makeTypedServiceCert(storage.ServiceType_SENSOR_SERVICE, sensorCert, sensorKey),
					makeTypedServiceCert(storage.ServiceType_COLLECTOR_SERVICE, collectorCert, collectorKey),
				},
			},
			expectedErr: true,
		},
		{
			description: "no service certificates",
			input: &storage.TypedServiceCertificateSet{
				CaPem: caCert,
			},
			expectedErr: true,
		},
		{
			description: "two certs",
			input: &storage.TypedServiceCertificateSet{
				CaPem: caCert,
				ServiceCerts: []*storage.TypedServiceCertificate{
					makeTypedServiceCert(storage.ServiceType_SENSOR_SERVICE, sensorCert, sensorKey),
					makeTypedServiceCert(storage.ServiceType_COLLECTOR_SERVICE, collectorCert, collectorKey),
				},
			},
			expectedFileMap: map[string][]byte{
				"ca-cert.pem":        caCert,
				"collector-cert.pem": collectorCert,
				"collector-key.pem":  collectorKey,
				"sensor-cert.pem":    sensorCert,
				"sensor-key.pem":     sensorKey,
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
			if c.expectedErr {
				assert.Error(t, err)
				assert.Nil(t, fileMap)
				return
			} else {
				assert.NoError(t, err)
			}
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
		input                              map[string][]byte
		expectedErr                        bool
		expectedUnknownServices            []string
		expectedTypedServiceCertificateSet *storage.TypedServiceCertificateSet
	}{
		{
			description:                        "empty",
			input:                              map[string][]byte{},
			expectedErr:                        true,
			expectedUnknownServices:            nil,
			expectedTypedServiceCertificateSet: nil,
		},
		{
			description: "missing CA",
			input: map[string][]byte{
				"collector-cert.pem": collectorCert,
				"collector-key.pem":  collectorKey,
			},
			expectedErr:                        true,
			expectedUnknownServices:            nil,
			expectedTypedServiceCertificateSet: nil,
		},
		{
			description: "known services",
			input: map[string][]byte{
				"ca-cert.pem":        caCert,
				"collector-cert.pem": collectorCert,
				"collector-key.pem":  collectorKey,
			},
			expectedErr:             false,
			expectedUnknownServices: nil,
			expectedTypedServiceCertificateSet: &storage.TypedServiceCertificateSet{
				CaPem: caCert,
				ServiceCerts: []*storage.TypedServiceCertificate{
					{
						ServiceType: storage.ServiceType_COLLECTOR_SERVICE,
						Cert: &storage.ServiceCertificate{
							CertPem: collectorCert,
							KeyPem:  collectorKey,
						},
					},
				},
			},
		},
		{
			description: "invalid key",
			input: map[string][]byte{
				"ca-cert.pem":        caCert,
				"collector-cert.pem": collectorCert,
				"collector-key.pem":  []byte("invalid key"),
			},
			expectedErr: true,
		},
		{
			description: "wrong-cert",
			input: map[string][]byte{
				"ca-cert.pem":        caCert,
				"collector-cert.pem": bogusCert,
				"collector-key.pem":  collectorKey,
			},
			expectedErr: true,
		},
		{
			description: "mixed known and unknown",
			input: map[string][]byte{
				"ca-cert.pem":        caCert,
				"collector-cert.pem": collectorCert,
				"collector-key.pem":  collectorKey,
				"foo-cert.pem":       []byte("cert 2"),
				"foo-key.pem":        []byte("key 2"),
				"bar-cert.pem":       []byte("cert 3"),
				"bar-key.pem":        []byte("key 3"),
			},
			expectedErr:             false,
			expectedUnknownServices: []string{"foo", "bar"},
			expectedTypedServiceCertificateSet: &storage.TypedServiceCertificateSet{
				CaPem: caCert,
				ServiceCerts: []*storage.TypedServiceCertificate{
					{
						ServiceType: storage.ServiceType_COLLECTOR_SERVICE,
						Cert: &storage.ServiceCertificate{
							CertPem: collectorCert,
							KeyPem:  collectorKey,
						},
					},
				},
			},
		},
		{
			description: "invalid file names",
			input: map[string][]byte{
				"collector-cert.pem": collectorCert,
				"collector.pem":      collectorKey,
			},
			expectedErr:             true,
			expectedUnknownServices: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.description, func(t *testing.T) {
			// Convert input FileMap into a TypedServiceCertificateSet.
			typedServiceCertificateSet, unknownServices, err := ConvertFileMapToTypedServiceCertificateSet(c.input)
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
				assert.Equal(t, c.input, roundTripFileMap)
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
