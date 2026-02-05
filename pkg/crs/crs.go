package crs

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/pkg/errors"
)

// CRS holds all core data which is required for a cluster registration secret.
type CRS struct {
	Version int      `json:"version"`
	CAs     []string `json:"CAs"`
	Cert    string   `json:"cert"`
	Key     string   `json:"key"`
}

// SerializeSecret serializes the given CRS into its opaque form.
func SerializeSecret(crs *CRS) (string, error) {
	if crs == nil {
		return "", errors.New("CRS is not initialized")
	}
	jsonSerialized, err := json.Marshal(crs)
	if err != nil {
		return "", errors.Wrap(err, "JSON marshalling CRS")
	}
	base64Encoded := base64.StdEncoding.EncodeToString(jsonSerialized)
	return base64Encoded, nil
}

// DeserializeSecret deserializes the opaque CRS.
func DeserializeSecret(serializedCrs string) (*CRS, error) {
	var deserializedCrs CRS
	base64Decoded, err := base64.StdEncoding.DecodeString(serializedCrs)
	if err != nil {
		return nil, errors.Wrap(err, "base64 decoding CRS")
	}
	err = json.Unmarshal(base64Decoded, &deserializedCrs)
	if err != nil {
		return nil, errors.Wrap(err, "JSON unmarshalling CRS")
	}
	if len(deserializedCrs.CAs) == 0 {
		return nil, errors.New("missing CA in CRS")
	}

	return &deserializedCrs, nil
}

// LoadFromFile loads an opaque CRS from the provided file.
func LoadFromFile(filePath string) (*CRS, error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "reading file %q", filePath)
	}
	if len(fileContent) == 0 {
		return nil, errors.New("CRS file is empty")
	}
	return DeserializeSecret(string(fileContent))
}

// Load loads an opaque CRS using environment settings given by mtls.CrsFilePathSetting.
func Load() (*CRS, error) {
	return LoadFromFile(crsFilePath())
}

// Certificate returns the X509 key pair contained in the CRS.
func (c *CRS) Certificate() (*tls.Certificate, error) {
	if c == nil {
		return nil, errors.New("CRS is not initialized")
	}
	cert, err := tls.X509KeyPair([]byte(c.Cert), []byte(c.Key))
	if err != nil {
		return nil, errors.Wrap(err, "parsing CRS certificate and key")
	}
	return &cert, nil
}

// CreateFakeCRS creates a fake CRS for testing purposes.
func CreateFakeCRS() *CRS {
	// Fake CA certificate - this is a self-signed certificate for testing
	fakeCA := `-----BEGIN CERTIFICATE-----
MIICljCCAX4CCQDKlQ+YHWXdozANBgkqhkiG9w0BAQsFADBNMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCVBhbG8gQWx0bzEOMAwGA1UECgwFRmFr
ZUNBMQ0wCwYDVQQDDARmYWtlMB4XDTIzMDEwMTAwMDAwMFoXDTI0MDEwMTAwMDAw
MFowTTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlQYWxvIEFs
dG8xDjAMBgNVBAoMBUZha2VDQTENMAsGA1UEAwwEZmFrZTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBALFY7H1j8z9+Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I
2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v
+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9
x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+
O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+
4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9
C+q5fwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBfL4oFq5VPi6m7oI2G9A6dT8fF
I6xdY7H+qH8pZ2Q1B3v4t5C6e8oP3Q2v9K7B1n4fH3dG2A5r7H1q3Z9I2fG8V0s
-----END CERTIFICATE-----`

	// Fake client certificate for the CRS
	fakeCert := `-----BEGIN CERTIFICATE-----
MIICljCCAX4CCQDKlQ+YHWXdpDANBgkqhkiG9w0BAQsFADBNMQswCQYDVQQGEwJV
UzELMAkGA1UECAwCQ0ExEjAQBgNVBAcMCVBhbG8gQWx0bzEOMAwGA1UECgwFRmFr
ZUNBMQ0wCwYDVQQDDARmYWtlMB4XDTIzMDEwMTAwMDAwMFoXDTI0MDEwMTAwMDAw
MFowTTELMAkGA1UEBhMCVVMxCzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlQYWxvIEFs
dG8xDjAMBgNVBAoMBUZha2VDQTENMAsGA1UEAwwEZmFrZTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBALFY7H1j8z9+Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I
2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v
+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9
x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+
O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+
4Q9V0v+T5QpH8v+O1YB5I2r9C+q5f9x2Q3K3H4+4Q9V0v+T5QpH8v+O1YB5I2r9
C+q5fwIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQBfL4oFq5VPi6m7oI2G9A6dT8fF
I6xdY7H+qH8pZ2Q1B3v4t5C6e8oP3Q2v9K7B1n4fH3dG2A5r7H1q3Z9I2fG8V0s
-----END CERTIFICATE-----`

	// Fake private key for the client certificate
	fakeKey := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsVjsfWPzP35DcrcfjqhD1XS/5PlCkfy/47VgHkjav0L6rl/3
HZDcrcfj7hD1XS/5PlCkfy/47VgHkjav0L6rl/3HZDcrcfj7hD1XS/5PlCkfy/4
7VgHkjav0L6rl/3HZDcrcfj7hD1XS/5PlCkfy/47VgHkjav0L6rl/3HZDcrcfj7
hD1XS/5PlCkfy/47VgHkjav0L6rl/3HZDcrcfj7hD1XS/5PlCkfy/47VgHkjav0
L6rl/3HZDcrcfj7hD1XS/5PlCkfy/47VgHkjav0L6rl/3HZDcrcfj7hD1XS/5Pl
Ckfy/47VgHkjav0L6rl/3HZDcrcfj7hD1XS/5PlCkfy/47VgHkjav0L6rl/wIDA
QABAoIBAQCw7dCJL7QkxAy+FG3k+2I1B8CeE2X7L5l+3o5z1P2f1u4a4g6h3x8c
s1G2s4t6p9I8V8v6c3K1w+M8l4G5o7t2K8Y9H4c3o7Y5L2e1Q9g8z3c7o8R4X5B
s8o1w8z4Y6E3f2z8Y3Q7h5A4q8F9r1Q8Y5n6I2o3K7G8f4H1z3g2Y9c8I7V8u2
y5E8X4q3I7w8m2Q5z1L3w9h8z4Y6K3g1F7u8H1m4n2I9s6v5k8o7Y3q8L2B9Y1
w8H5t3z2s6Y1L8o3F7Y9s8P2X4z3q8Y9H5c1I8K7w2B8t4z6L1Y8o3n9K2z1G8
h4Y6q3I7s8F2z1A9o7K8z3Y6L2X4s8h1G9o2z8H3q1r1T5z6J9Y3q8L2B9Y1w8
H5t3z2s6Y1L8o3F7Y9s8P2X4z3q8Y9H5c1AoGBANXX1g8H5B2o7Y3q8L2B9Y1w8
H5t3z2s6Y1L8o3F7Y9s8P2X4z3q8Y9H5c1I8K7w2B8t4z6L1Y8o3n9K2z1G8h4
Y6q3I7s8F2z1A9o7K8z3Y6L2X4s8h1G9o2z8H3q1r1T5z6J9Y3q8L2B9Y1w8H5
t3z2s6Y1L8o3F7Y9s8P2X4z3q8Y9H5c1AoGBANXX1g8H5B2o7Y3q8L2B9Y1w8H5
t3z2s6Y1L8o3F7Y9s8P2X4z3q8Y9H5c1I8K7w2B8t4z6L1Y8o3n9K2z1G8h4Y6
q3I7s8F2z1A9o7K8z3Y6L2X4s8h1G9o2z8H3q1r1T5z6J9Y3q8L2B9Y1w8H5t3
z2s6Y1L8o3F7Y9s8P2X4z3q8Y9H5c1
-----END RSA PRIVATE KEY-----`

	return &CRS{
		Version: 1,
		CAs:     []string{fakeCA},
		Cert:    fakeCert,
		Key:     fakeKey,
	}
}
