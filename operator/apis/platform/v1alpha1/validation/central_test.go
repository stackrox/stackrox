package validation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func TestValidateCentral(t *testing.T) {

	type testCase struct {
		name    string
		central v1alpha1.Central
		assert  func(t *testing.T, errorList field.ErrorList)
	}
	certChain1With1Cert, err := createCertificateChain(1)
	require.NoError(t, err)

	chain1RootCA := string(certChain1With1Cert[0])

	certChain2With2Certs, err := createCertificateChain(2)
	require.NoError(t, err)

	chain2RootCA := string(certChain2With2Certs[1])
	chain2IntermediateCA := string(certChain2With2Certs[0])

	certChainWith3Certs, err := createCertificateChain(3)
	require.NoError(t, err)

	chain3RootCA := string(certChainWith3Certs[2])
	chain3IntermediateCA := string(certChainWith3Certs[1])
	chain3LeafCA := string(certChainWith3Certs[0])

	testCases := []testCase{
		{
			name: "spec.tls: should not panic if tls is nil",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: nil,
				},
			},
			assert: func(t *testing.T, errorList field.ErrorList) {
				assert.Empty(t, errorList)
			},
		}, {
			name: "spec.tls.additionalCAs: name must be provided",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Content: chain1RootCA,
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Required(
					field.NewPath("spec.tls.additionalCAs[0].name"),
					errAdditionalCANameRequired,
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: names must be unique",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name:    "name",
								Content: chain1RootCA,
							}, {
								Name:    "name",
								Content: chain2RootCA,
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Duplicate(
					field.NewPath("spec.tls.additionalCAs[1].name"),
					"name",
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: content must be provided",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Required(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					errAdditionalCAContentRequired,
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: content must be valid PEM",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name:    "name",
								Content: "invalid",
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Invalid(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					"invalid",
					"failed to parse the provided CA cert content: failed to parse 1st certificate in chain: no PEM data found",
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: PEM headers must be of type CERTIFICATE",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								Content: strings.Join([]string{
									chain1RootCA,
									"-----BEGIN SNOOPY-----\n-----END SNOOPY-----",
								}, "\n"),
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Invalid(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					strings.Join([]string{
						chain1RootCA,
						"-----BEGIN SNOOPY-----\n-----END SNOOPY-----",
					}, "\n"),
					"failed to parse the provided CA cert content: failed to parse 2nd certificate in chain: unexpected PEM type 'SNOOPY'",
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: PEM content must be a valid certificate",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								Content: strings.Join([]string{
									chain1RootCA,
									"-----BEGIN CERTIFICATE-----\naW52YWxpZAo=\n-----END CERTIFICATE-----",
								}, "\n"),
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Invalid(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					strings.Join([]string{
						chain1RootCA,
						"-----BEGIN CERTIFICATE-----\naW52YWxpZAo=\n-----END CERTIFICATE-----",
					}, "\n"),
					"failed to parse the provided CA cert content: failed to parse 2nd certificate in chain: x509: malformed certificate",
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: PEM content must be base64-encoded",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								Content: strings.Join([]string{
									chain1RootCA,
									"-----BEGIN CERTIFICATE-----\ninvalid!\n-----END CERTIFICATE-----",
								}, "\n"),
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Invalid(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					strings.Join([]string{
						chain1RootCA,
						"-----BEGIN CERTIFICATE-----\ninvalid!\n-----END CERTIFICATE-----",
					}, "\n"),
					"failed to parse the provided CA cert content: failed to parse 2nd certificate in chain: no PEM data found",
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: the certificate chain must be valid",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								// Concatenating two unrelated certificate chains
								Content: strings.Join([]string{
									chain1RootCA,
									chain2IntermediateCA,
								}, "\n"),
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Invalid(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					strings.Join([]string{
						chain1RootCA,
						chain2IntermediateCA,
					}, "\n"),
					`failed to verify the certificate chain: could not verify the 2nd certificate in the chain: x509: certificate signed by unknown authority (possibly because of "crypto/rsa: verification error" while trying to verify candidate authority certificate "RHACS")`,
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: the certificate chain must be in order",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								// Reversing the order of the certificate chain
								Content: strings.Join([]string{
									chain3LeafCA,
									chain3RootCA,
									chain3IntermediateCA,
								}, "\n"),
							},
						},
					},
				},
			},
			assert: errorsEqual(field.ErrorList{
				field.Invalid(
					field.NewPath("spec.tls.additionalCAs[0].content"),
					strings.Join([]string{
						chain3LeafCA,
						chain3RootCA,
						chain3IntermediateCA,
					}, "\n"),
					`failed to verify the certificate chain: could not verify the 3rd certificate in the chain: x509: certificate signed by unknown authority (possibly because of "crypto/rsa: verification error" while trying to verify candidate authority certificate "RHACS")`,
				),
			}),
		}, {
			name: "spec.tls.additionalCAs: using a single certificate is allowed",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name:    "name",
								Content: chain1RootCA,
							},
						},
					},
				},
			},
			assert: func(t *testing.T, errorList field.ErrorList) {
				assert.Empty(t, errorList)
			},
		}, {
			name: "additionalCAs: using two certificates is allowed",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								Content: strings.Join([]string{
									chain2IntermediateCA,
									chain2RootCA,
								}, "\n"),
							},
						},
					},
				},
			},
			assert: func(t *testing.T, errorList field.ErrorList) {
				assert.Empty(t, errorList)
			},
		}, {
			name: "spec.tls.additionalCAs: using multiple certificates is allowed",
			central: v1alpha1.Central{
				Spec: v1alpha1.CentralSpec{
					TLS: &v1alpha1.TLSConfig{
						AdditionalCAs: []v1alpha1.AdditionalCA{
							{
								Name: "name",
								Content: strings.Join([]string{
									chain3LeafCA,
									chain3IntermediateCA,
									chain3RootCA,
								}, "\n"),
							},
						},
					},
				},
			},
			assert: func(t *testing.T, errorList field.ErrorList) {
				assert.Empty(t, errorList)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, ValidateCentral(&tc.central))
		})
	}
}

func errorsEqual(errors field.ErrorList) func(t *testing.T, errorList field.ErrorList) {
	return func(t *testing.T, errorList field.ErrorList) {
		assert.Equalf(t, errors, errorList, "expected %v but got %v", errors, errorList)
	}
}

func createCertificateChain(length int) ([][]byte, error) {

	if length < 1 {
		return [][]byte{}, nil
	}

	subject := pkix.Name{
		Organization:  []string{"RHACS"},
		Country:       []string{"US"},
		Locality:      []string{"San Francisco"},
		StreetAddress: []string{"Golden Gate Bridge"},
		PostalCode:    []string{"94016"},
	}

	caSerialNumber, err := generateSerialNumber()
	if err != nil {
		return nil, err
	}

	ca := &x509.Certificate{
		SerialNumber:          caSerialNumber,
		Subject:               subject,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	caCert, err := x509.ParseCertificate(caBytes)
	if err != nil {
		return nil, err
	}

	caPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	result := [][]byte{caPEM}

	var currentParent = caCert
	var currentParentPrivKey = caPrivKey

	for i := 1; i < length; i++ {

		childSerialNumber, err := generateSerialNumber()
		if err != nil {
			return nil, err
		}

		childCertificateTemplate := &x509.Certificate{
			SerialNumber:          childSerialNumber,
			Subject:               subject,
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			IsCA:                  true,
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			BasicConstraintsValid: true,
		}

		// create our private and public key
		childPrivateKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return nil, err
		}

		// create the intermediate CA
		childCertificateBytes, err := x509.CreateCertificate(rand.Reader, childCertificateTemplate, currentParent, &childPrivateKey.PublicKey, currentParentPrivKey)
		if err != nil {
			return nil, err
		}

		childCertificate, err := x509.ParseCertificate(childCertificateBytes)
		if err != nil {
			return nil, err
		}

		childCertificatePem := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: childCertificateBytes,
		})

		result = append(result, childCertificatePem)
		currentParent = childCertificate
		currentParentPrivKey = childPrivateKey

	}

	// reverse the order of the certificates
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

func generateSerialNumber() (*big.Int, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	return rand.Int(rand.Reader, serialNumberLimit)
}
