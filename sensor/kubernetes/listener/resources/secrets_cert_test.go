package resources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateCertPEM(t *testing.T, template *x509.Certificate) string {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER}))
}

func TestConvertPkixName(t *testing.T) {
	cases := map[string]struct {
		name             pkix.Name
		wantCommonName   string
		wantCountry      string
		wantOrg          string
		wantNamesContain string
	}{
		"simple name": {
			name: pkix.Name{
				CommonName:   "example.com",
				Country:      []string{"US"},
				Organization: []string{"Example Inc"},
			},
			wantCommonName: "example.com",
			wantCountry:    "US",
			wantOrg:        "Example Inc",
		},
		"multi-value country": {
			name: pkix.Name{
				CommonName: "multi",
				Country:    []string{"US", "UK"},
			},
			wantCommonName: "multi",
			wantCountry:    "US, UK",
		},
		"multi-value org": {
			name: pkix.Name{
				Organization: []string{"Org1", "Org2"},
			},
			wantOrg: "Org1, Org2",
		},
		"names contains raw values": {
			name: pkix.Name{
				CommonName: "test-cn",
				Names: []pkix.AttributeTypeAndValue{
					{Type: asn1.ObjectIdentifier{2, 5, 4, 3}, Value: "test-cn"},
					{Type: asn1.ObjectIdentifier{2, 5, 4, 10}, Value: "test-org"},
				},
			},
			wantCommonName:   "test-cn",
			wantNamesContain: "test-cn",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := convertPkixName(tc.name)
			assert.Equal(t, tc.wantCommonName, result.GetCommonName())
			if tc.wantCountry != "" {
				assert.Equal(t, tc.wantCountry, result.GetCountry())
			}
			if tc.wantOrg != "" {
				assert.Equal(t, tc.wantOrg, result.GetOrganization())
			}
			if tc.wantNamesContain != "" {
				assert.Contains(t, result.GetNames(), tc.wantNamesContain)
			}
		})
	}
}

func TestConvertPkixNameUsesValueNotStruct(t *testing.T) {
	name := pkix.Name{
		Names: []pkix.AttributeTypeAndValue{
			{Type: asn1.ObjectIdentifier{2, 5, 4, 3}, Value: "My CN"},
		},
	}
	result := convertPkixName(name)
	assert.Equal(t, []string{"My CN"}, result.GetNames())
}

func TestParseCertData(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "test.example.com",
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
		},
		Issuer: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:          now.Add(-time.Hour),
		NotAfter:           now.Add(24 * time.Hour),
		SignatureAlgorithm: x509.ECDSAWithSHA256,
		DNSNames:           []string{"test.example.com", "*.example.com"},
		IPAddresses:        []net.IP{net.ParseIP("10.0.0.1")},
		EmailAddresses:     []string{"admin@example.com"},
		URIs:               []*url.URL{{Scheme: "spiffe", Host: "cluster.local", Path: "/ns/default/sa/test"}},
	}
	certPEM := generateCertPEM(t, template)

	cert := parseCertData(certPEM)
	require.NotNil(t, cert)

	assert.Equal(t, "test.example.com", cert.GetSubject().GetCommonName())
	assert.Equal(t, "US", cert.GetSubject().GetCountry())
	assert.Equal(t, "Test Org", cert.GetSubject().GetOrganization())

	assert.Equal(t, "ECDSA-SHA256", cert.GetAlgorithm())

	expectedSANs := []string{
		"test.example.com",
		"*.example.com",
		"admin@example.com",
		"10.0.0.1",
		"spiffe://cluster.local/ns/default/sa/test",
	}
	assert.ElementsMatch(t, expectedSANs, cert.GetSans())

	assert.NotNil(t, cert.GetStartDate())
	assert.NotNil(t, cert.GetEndDate())
}

func TestParseCertDataInvalidInput(t *testing.T) {
	cases := map[string]struct {
		input string
	}{
		"garbage": {input: "not a cert"},
		"empty":   {input: ""},
		"bad DER": {input: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("bad")}))},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Nil(t, parseCertData(tc.input))
		})
	}
}
