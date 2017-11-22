package keys

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"reflect"
	"testing"
)

type keyTestCase struct {
	name          string
	block         []byte
	expectedError error
	keyType       KeyType
}

var (
	baseCert = []byte(`-----BEGIN CERTIFICATE-----
MIIFdTCCA12gAwIBAgIJALJpOl/zFWYaMA0GCSqGSIb3DQEBCwUAME8xCzAJBgNV
BAYTAkZSMQ4wDAYDVQQHDAVQYXJpczEOMAwGA1UECgwFRWtpbm8xDzANBgNVBAsM
BkRldk9wczEPMA0GA1UEAwwGbXljZXJ0MB4XDTE2MDgyOTE4MzMwMFoXDTI2MDgy
NzE4MzMwMFowTzELMAkGA1UEBhMCRlIxDjAMBgNVBAcMBVBhcmlzMQ4wDAYDVQQK
DAVFa2lubzEPMA0GA1UECwwGRGV2T3BzMQ8wDQYDVQQDDAZteWNlcnQwggIiMA0G
CSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCUTaySQhEbepDK+VHZqfslAsuoJS4s
VS5WIiquMUnPOo0dZw70M2CBkN2b26MAwGKM+sU+xYqUSsHKweyxCvXOfaJFgPv6
C7v4en4fLeQUstERbAe5x13WTgcbleXYNIYRL3Nn5dwtNZSKnORFBceK6N0Ub048
6Ar/3UXrYLgV+drX92JQbaaqNtf3bx/lskc4Pz6sCYJ5Ghfeb50R9vYWUaTD38jp
bWZFvwCp61Aq4gzMXKQWUEpYOvE1DtOA3ewlBV1K7yhG+cJJH8a5i4khUfUUUlpI
EotJipQlvZueijEGP8LCCQQPMZcx9m+gF5SOEkU0walFGFza7ceBsa6lMYE6JN9P
v4X81WeKVdALVZpot3U7YPmKL0fdW/iZf9BvhQokdMWwpbocqQWtBROLfFcEqOW1
M20LlIRPAwM9Chy7pRbLLovx75Y3cPjkJhkmqKvVMEnHQ+LccYHBUvTreaxigKaZ
fxt6TX/84PRg28l0B0bi9WAXOOeBewQBkk/D/nEE46mZrRARRWMXB4VhA6HTFp8d
FcZElewMzZVC29u3YHhjxUjjjtusk7ff72wPx+TSKyLO9LjadpzPQWnhJtX9u2ah
WsNi9pEjJeWyQeYTiURztOUvUcv0CEmt4+QeHz6vUPCan2KpYQwilU32Qipw7vZ4
zfLCleWUMqGwiQIDAQABo1QwUjASBgNVHRMBAf8ECDAGAQH/AgEAMDwGA1UdEQQ1
MDOCEGxvZ3MuZXhhbXBsZS5jb22CE21ldHJpY3MuZXhhbXBsZS5jb22HBMCoAAGH
BAoAADIwDQYJKoZIhvcNAQELBQADggIBAFuHB96q2LkoHZsxGrOnSHYdGTaN7UzT
PgmwlvnPEw2bf7jKQIh256tGa7iRf6iMNRRFxQx9Rg8jB7WtegmvZLokZF1wvmgu
aihd82Rdy/BNEnx8kMvRIMtJcTwXeCW8xWVjwefVNKsW5Ti05CQYMGyQQD7xC4in
2D71E2OS5PI5wmJCEaqEa96VWXlYVJW5YWJBDhl5M7LYsDGPrqw21SBLHUl3YkLy
yOyBOWV1RwLOS6zCkuQ6n0OYl8QoDvD4kW9dtv9zaIY3YGX6tshwbGmPYuIqIveX
aPcHwW3TkFkJm9Tyl7C07ObwVj7yjrwe4Esstss7sL7fHH911znPyOdy7UjyW1Y7
A2MvKgBfKbd82MkLbIae7XTmwbmEmM/ma2dNfImPd8gIbOnNWdSGRM3RzTZTfu8F
DnQrkUOOS+nAHYR6wwjBm5te4x6CqLm2oyWU3EumAfEfe1plCeZvX+KzL1WD4N8v
9mD+kRbBW/UQoP05AW8iIqF+37wMA3hGGIHIb2zbKFe+qbT+IkDVPacD/1zn6ITH
Usfysk5Vdv+5goBI4gz+ATMEQsHlFKSYHuIISLxmsHWZD37JRPfsQaml6lwbwmJl
L4lXqMjlLGAuImvYLMbNDQnmaujbC2brKX70kFOwYTQFuEGObh1us3J/QTqHUNmT
TT/g/1zcSxxc
-----END CERTIFICATE-----`)
	certBlock, _ = pem.Decode(baseCert)

	basePrivateKey = []byte(
		`-----BEGIN PRIVATE KEY-----
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
VuNdpypvyoWRxuBixzD2S3Kpiccb9w==
-----END PRIVATE KEY-----`)
	keyBlock, _ = pem.Decode(basePrivateKey)

	certTestCases = []keyTestCase{
		{
			name:          `PEM (raw base cert)`,
			block:         baseCert,
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `PEM no linebreaks`,
			block:         bytes.Replace(baseCert, []byte("\n"), []byte(""), -1),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `PEM linebreaks as spaces`,
			block:         bytes.Replace(baseCert, []byte("\n"), []byte(" "), -1),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `PEM linebreaks as tabs`,
			block:         bytes.Replace(baseCert, []byte("\n"), []byte("\t"), -1),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `PEM with no begin header`,
			block:         bytes.Replace(baseCert, []byte("-----BEGIN CERTIFICATE-----\n"), []byte(""), -1),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `PEM with no end header`,
			block:         bytes.Replace(baseCert, []byte("\n-----END CERTIFICATE-----"), []byte(""), -1),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `Base64 PEM`,
			block:         []byte(base64.StdEncoding.EncodeToString(baseCert)),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `Base64 Base64 PEM`,
			block:         []byte(base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(baseCert)))),
			expectedError: errUnknownCertFormat,
			keyType:       Public,
		},
		{
			name:          `DER`,
			block:         certBlock.Bytes,
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `Base64 DER`,
			block:         []byte(base64.StdEncoding.EncodeToString(certBlock.Bytes)),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `Base64 Base64 DER`,
			block:         []byte(base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(certBlock.Bytes)))),
			expectedError: nil,
			keyType:       Public,
		},
		{
			name:          `Empty PEM`,
			block:         pem.EncodeToMemory(&pem.Block{}),
			expectedError: errEmptyCert,
			keyType:       Public,
		},
		{
			name:          `Base64 Empty PEM`,
			block:         []byte(base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{}))),
			expectedError: errEmptyCert,
			keyType:       Public,
		},
		{
			name:          `gibberish`,
			block:         []byte(`asdfasdf`),
			expectedError: errUnknownCertFormat,
			keyType:       Public,
		},
		{
			name:          `base64 gibberish`,
			block:         []byte(base64.StdEncoding.EncodeToString([]byte(`asdfasdf`))),
			expectedError: errUnknownCertFormat,
			keyType:       Public,
		},
		{
			name:          `pem gibberish`,
			block:         pem.EncodeToMemory(&pem.Block{Type: `CERTIFICATE`, Bytes: []byte(`asdfasdf`)}),
			expectedError: errUnknownCertFormat,
			keyType:       Public,
		},
		{
			name:          `wrong format`,
			block:         certBlock.Bytes,
			expectedError: errUnknownEncoding,
			keyType:       Private,
		},
	}
	keyTestCases = []keyTestCase{
		{
			name:          `key PEM`,
			block:         basePrivateKey,
			expectedError: nil,
			keyType:       Private,
		},
		{
			name:          `key Base64 PEM`,
			block:         []byte(base64.StdEncoding.EncodeToString(basePrivateKey)),
			expectedError: nil,
			keyType:       Private,
		},
		{
			name:          `key Base64 Base64 PEM`,
			block:         []byte(base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(basePrivateKey)))),
			expectedError: errUnknownKeyFormat,
			keyType:       Private,
		},
		{
			name:          `key DER`,
			block:         keyBlock.Bytes,
			expectedError: nil,
			keyType:       Private,
		},
		{
			name:          `key Base64 DER`,
			block:         []byte(base64.StdEncoding.EncodeToString(keyBlock.Bytes)),
			expectedError: nil,
			keyType:       Private,
		},
		{
			name:          `key Base64 Base64 DER`,
			block:         []byte(base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(keyBlock.Bytes)))),
			expectedError: errUnknownKeyFormat,
			keyType:       Private,
		},
		{
			name:          `key Empty PEM`,
			block:         pem.EncodeToMemory(&pem.Block{}),
			expectedError: errEmptyKey,
			keyType:       Private,
		},
		{
			name:          `key Base64 Empty PEM`,
			block:         []byte(base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{}))),
			expectedError: errEmptyKey,
			keyType:       Private,
		},
		{
			name:          `key gibberish`,
			block:         []byte(`asdfasdf`),
			expectedError: errUnknownKeyFormat,
			keyType:       Private,
		},
		{
			name:          `key base64 gibberish`,
			block:         []byte(base64.StdEncoding.EncodeToString([]byte(`asdfasdf`))),
			expectedError: errUnknownKeyFormat,
			keyType:       Private,
		},
		{
			name:          `key pem gibberish`,
			block:         pem.EncodeToMemory(&pem.Block{Type: `PRIVATE KEY`, Bytes: []byte(`asdfasdf`)}),
			expectedError: errUnknownKeyFormat,
			keyType:       Private,
		},
		{
			name:          `key wrong format`,
			block:         basePrivateKey,
			expectedError: errUnknownCertFormat,
			keyType:       Public,
		},
	}
)

func TestKeyChecks(t *testing.T) {
	t.Parallel()

	for _, test := range append(certTestCases, keyTestCases...) {
		var err error
		if test.keyType == Public {
			_, err = NewCertificate(string(test.block))
		} else if test.keyType == Private {
			_, err = NewPrivateKey(string(test.block))
		}
		if err != test.expectedError {
			t.Errorf("Unexpected error in test case: %s", test.name)
			t.Logf("Expected %v; Received %v", test.expectedError, err)
		}
	}
}

func TestGeneratedKeysValidate(t *testing.T) {
	t.Parallel()

	//generate RSA private key
	rsaKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		t.Error(err)
	}
	//marshal to DER
	derRSA := x509.MarshalPKCS1PrivateKey(rsaKey)
	_, err = NewPrivateKey(string(derRSA))
	//verify that it unmarshals from DER correctly
	keyCheck, err := x509.ParsePKCS1PrivateKey(derRSA)
	if err != nil {
		t.Error(err)
	}
	//verify pre-marshal and unmarshal are equal
	if !reflect.DeepEqual(rsaKey, keyCheck) {
		t.Error("key check failed")
	}

	//verify construction from rsa DER
	_, err = NewPrivateKey(string(derRSA))
	if err != nil {
		t.Error(err)
	}

	//verify construction from PEM rsa DER
	pemRSA := pem.EncodeToMemory(&pem.Block{Type: `RSA PRIVATE KEY`, Bytes: derRSA})
	_, err = NewPrivateKey(string(pemRSA))
	if err != nil {
		t.Error(err)
	}

	//various forms of ECDSA keys
	ecdsaKey, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		t.Error(err)
	}
	derECDSA, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		t.Error(err)
	}
	_, err = NewPrivateKey(string(derECDSA))
	if err != nil {
		t.Error(err)
	}
	pemECDSA := pem.EncodeToMemory(&pem.Block{Type: `ECDSA PRIVATE KEY`, Bytes: derECDSA})
	_, err = NewPrivateKey(string(pemECDSA))
	if err != nil {
		t.Error(err)
	}
}
