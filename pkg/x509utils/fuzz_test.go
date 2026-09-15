package x509utils

import (
	"bytes"
	"encoding/pem"
	"testing"
)

// FuzzConvertPEMToDERs fuzzes the ConvertPEMToDERs function with arbitrary bytes.
// This tests the PEM decoder's ability to handle malformed input without panicking.
func FuzzConvertPEMToDERs(f *testing.F) {
	// Seed corpus with valid PEM data from existing test data
	f.Add([]byte(pemChain))

	// Seed with individual certificates extracted from chain
	f.Add([]byte(`-----BEGIN CERTIFICATE-----
MIIFCDCCAvCgAwIBAgIJAJsvSyhFXCV2MA0GCSqGSIb3DQEBCwUAMB8xHTAbBgNV
BAMMFEludGVybWVkaWF0ZSBUZXN0IENBMB4XDTIyMDkwMzExNTczN1oXDTIyMTAw
MzExNTczN1owMTEvMC0GA1UEAwwmY3VzdG9tLXRscy1jZXJ0LmNlbnRyYWwuc3Rh
Y2tyb3gubG9jYWwwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQCunJqv
R/LRGISE+oG52Yify9II/y7U7kcTwTufkU/RYcsGuoXtMdSh1b3Poi8kVHTQXQ/+
4VOt9jCL9R7+sIwjilIE2gd8rsW70bN33k3RhhNF/W/vwMeajcLgcM/fJt3jRztN
MLjVlX6061TLcy9aI8gFwsoBi1ZCHhkB8mv9YQNDWG0RtsksNT0ltWiIsowrn0Ol
+vvrohQdtuets1rMTE7m8LPUeYoPVzdsf1VbG7klrkc64KtIkkINQ+hWZEpFx2eL
b5mkrSVrVTFVIz2G+J1iOaMiXLQnjbEERqOvy9OMZtiivzj92O0O4DB/qNUhxEpz
mqTzfGgWHhVUXeZtSGHS02WfD1hVaoZSWxSCASk1YciZ1i//XKlH3rmtkMDOObpQ
E+AnNWH5e2oX9WNfV0SMr36yz8bMtsutPTGcSiO3Z7YsZMHSNJPR433uCUddANAr
Gv8KTtFiweeXrozNDaWexXfklxQxAlqDZYnt+tQ7NhBZzkPgzOn7usDTCChQzzoE
Fa5L3PDjzkL7wwxHsKUurxvy6Y1nti80Ls7yfA8o22sD7ZGptjlkkmi6p/SpynWr
jsTeGhrasru0w1BK8B90cmufFH+U8NPky5mn1n4Tsf/1F22sg3GqbG+Vw9InCgZC
ZGX/NiZgbAdp7H6yZ29vwK3zfRNLRvqjQJOtPQIDAQABozUwMzAxBgNVHREEKjAo
giZjdXN0b20tdGxzLWNlcnQuY2VudHJhbC5zdGFja3JveC5sb2NhbDANBgkqhkiG
9w0BAQsFAAOCAgEAhJtf/sTrCcb8NTpI8LOJXNsnPogXXGr5+3C3OvN4VQ5Gl10R
AJf4c9hVQ80FhBXgvK4oO5ADEdJhJpsQSM2r7MxWiJWd+yBzRk/GGP7x4aC5IG3k
pALHZQCyIQwm3Di0djVKJ+rJGV5edsDLahqigOVAe9arquhC75KfVHoi1xNYX3Ex
9vFZOnYkpNybk/wvoa67AbzzKRUtq0uc167hLqjCsq4bZzPrvx9hqsOpcmVA1iW4
LIJ3V7Z143Dd4bp56hCXZ3vl0YHqojEvlwYBalTgnNm0IwUOTFQO5q0PkzJfAvTS
7hvJUeSe9y6GnGP2v5+r+VxkaFoE4rou7n5s6L3uaQpvRDzRqeYLbG9iFfc21cbN
bGmFL9RD4v/LbsqZgeELbOo6bDfeRH/zyFA0CGe6siMNvW+Y1ziTfkIJiFKzqdyb
Vwm3ZCq/vZcKFOD3YooHgRxTJCMO2BlYnaNACh0xE6PvvpgNCUtOkOsCm06d59Rz
VVUvSXRjc1KBJ58sMMi766KOOeVlbENLiGNMqPyaaqFYKxEkIxcfNQzvALh09QM0
ODlUGgydiyT2g4SP7xiyq6lbaq9OSj7vdRd+CK8CXEDOOHRDnmlswICgp0h62rvU
hPVFPjweENJXYjHAvLbaUqKNIqkWPeuViPbUDBrxkdNsd/6P9s7k+Xcr31I=
-----END CERTIFICATE-----`))

	// Seed with empty input
	f.Add([]byte{})

	// Seed with non-PEM data
	f.Add([]byte("not a PEM certificate"))

	// Seed with PEM header but no content
	f.Add([]byte("-----BEGIN CERTIFICATE-----\n-----END CERTIFICATE-----"))

	// Seed with malformed PEM
	f.Add([]byte("-----BEGIN CERTIFICATE-----\ninvalid base64 !@#$%\n-----END CERTIFICATE-----"))

	// Seed with valid PEM structure but invalid data
	invalidPEMBuf := bytes.NewBuffer(nil)
	_ = pem.Encode(invalidPEMBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte{0x00, 0x01, 0x02},
	})
	f.Add(invalidPEMBuf.Bytes())

	// Seed with multiple concatenated PEMs (some valid, some invalid)
	f.Add([]byte("-----BEGIN CERTIFICATE-----\nYWJjZA==\n-----END CERTIFICATE-----\n-----BEGIN CERTIFICATE-----\nYWJjZA==\n-----END CERTIFICATE-----"))

	f.Fuzz(func(t *testing.T, data []byte) {
		// The function should never panic, regardless of input
		_, _ = ConvertPEMToDERs(data)
		// Note: We don't assert on errors since malformed input is expected to error
	})
}

// FuzzConvertPEMTox509Certs fuzzes the ConvertPEMTox509Certs function with arbitrary bytes.
// This tests both PEM decoding and X.509 certificate parsing without panicking.
func FuzzConvertPEMTox509Certs(f *testing.F) {
	// Seed corpus with valid PEM chain
	f.Add([]byte(pemChain))

	// Seed with single valid certificate
	f.Add([]byte(`-----BEGIN CERTIFICATE-----
MIICxDCCAaygAwIBAgIJALXVoHYCZjlEMA0GCSqGSIb3DQEBCwUAMBcxFTATBgNV
BAMMDFJvb3QgVGVzdCBDQTAeFw0yMjA5MDMxMTU3MzZaFw0yMjEwMDMxMTU3MzZa
MBcxFTATBgNVBAMMDFJvb3QgVGVzdCBDQTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAM97BXRilwaeEbj4ydoxz5VlqPH9ksq2tod//tErLfLQKh28u54M
H1CEW+PGjo2KP4zaDNM4CGPo3sMTMyltHu+vF59wW7rZHTnfka6bRONREz1vJ5GR
cLHfqehRsuBr56dacupD65V2NOMFr/h9/QznwXP8MPt6WFV1qyDonjXrbXtesIRq
TfHs++dPQHtarOaHlPhImONHL/Z+Lw7TWdCoFxSIXVwEcQT9NE9xP4cnnIa15h+Q
YhgYFsnl4KxhGblzeD8WJjbvG+/7UI6WLYc7W8OmKg8iFbB2jNlAoQ4nT2ikn8FD
7gAUXcfiq7uNlUy28QfAByZMlRWyFv373a0CAwEAAaMTMBEwDwYDVR0TBAgwBgEB
/wIBATANBgkqhkiG9w0BAQsFAAOCAQEANUv8f2SPfLShoBHvUsihpTu5tWmw/0Lv
XVUWlA55oRIwmm04vsb29z5WwMhQEUtsWAzhEmVq6wRgRpvBr21ZNkm2jztMuaeT
jA9B3yIiiOtzOaUCNwT5TaBUDp8OTzzv51+d63IC1k6bRAd1yPR0HjbeJEkyO4Lj
nuSJQs60mOHpxyn45+loo6rvV6AKrnNH/O6Lh8CyBvOhg9qVgSQc9yQlfuEuB7BH
n7hvqh4tgXMmdviwLScB4K1EJucP1O20IkWN1tCBdZRctvxDRBU7cMYvE0ADAv11
lnNKB0G/yk2nCUoPH6OspqsEfr4my+8F4bQBtavbflWUjqUiSHjodQ==
-----END CERTIFICATE-----`))

	// Seed with empty input
	f.Add([]byte{})

	// Seed with non-PEM data
	f.Add([]byte("random text that is not a certificate"))

	// Seed with valid PEM structure but invalid X.509 data
	invalidX509PEMBuf := bytes.NewBuffer(nil)
	_ = pem.Encode(invalidX509PEMBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: make([]byte, 0),
	})
	f.Add(invalidX509PEMBuf.Bytes())

	// Seed with valid PEM but corrupted certificate data
	corruptedPEMBuf := bytes.NewBuffer(nil)
	_ = pem.Encode(corruptedPEMBuf, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: []byte{0x30, 0x82, 0x01, 0x00}, // Valid ASN.1 header but truncated
	})
	f.Add(corruptedPEMBuf.Bytes())

	// Seed with PEM containing non-certificate types
	nonCertPEMBuf := bytes.NewBuffer(nil)
	_ = pem.Encode(nonCertPEMBuf, &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: []byte{0x30, 0x82, 0x01, 0x00},
	})
	f.Add(nonCertPEMBuf.Bytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		// The function should never panic, regardless of input
		_, _ = ConvertPEMTox509Certs(data)
		// Note: We don't assert on errors since malformed input is expected to error
	})
}

// FuzzParseCertificateChain fuzzes the ParseCertificateChain function.
// Since it takes a slice of byte slices, we need to construct valid-looking DER data.
func FuzzParseCertificateChain(f *testing.F) {
	// Seed corpus with valid DER data from existing test
	if len(derChain) > 0 {
		f.Add(derChain[0])
	}
	if len(derChain) > 1 {
		f.Add(derChain[1])
	}
	if len(derChain) > 2 {
		f.Add(derChain[2])
	}

	// Seed with empty DER
	f.Add([]byte{})

	// Seed with random bytes
	f.Add([]byte{0x00, 0x01, 0x02, 0x03})

	// Seed with valid ASN.1 SEQUENCE header but no content
	f.Add([]byte{0x30, 0x00})

	// Seed with truncated ASN.1 structure
	f.Add([]byte{0x30, 0x82, 0x01, 0x00})

	// Seed with valid-looking but corrupted certificate start
	f.Add([]byte{0x30, 0x82, 0x03, 0x00, 0xa0, 0x03, 0x02, 0x01})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Test with single DER certificate
		chain := [][]byte{data}
		_, _ = ParseCertificateChain(chain)

		// Test with multiple copies of the same data
		multiChain := [][]byte{data, data, data}
		_, _ = ParseCertificateChain(multiChain)

		// Test with empty chain
		_, _ = ParseCertificateChain([][]byte{})

		// Note: We don't assert on errors since malformed input is expected to error
		// The key is that the function should never panic
	})
}
