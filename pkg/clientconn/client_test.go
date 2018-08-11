package clientconn

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"strconv"
	"testing"
	"time"

	"github.com/cloudflare/cfssl/config"
	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/initca"
	cfsigner "github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type generatedCertOpts struct {
	expired       bool
	cn            string
	invalidRootCA bool
	useWrongKey   bool
}

func generateCACert(t *testing.T) (parentKey *ecdsa.PrivateKey, parent *x509.Certificate, parentCert []byte) {
	parentKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	req := cfcsr.CertificateRequest{
		CN:         "StackRox Prevent Certificate Authority",
		KeyRequest: cfcsr.NewBasicKeyRequest(),
	}
	parentCertPEM, _, err := initca.NewFromSigner(&req, parentKey)
	require.NoError(t, err)

	decoded, _ := pem.Decode(parentCertPEM)
	require.NotNil(t, decoded)

	parentCert = decoded.Bytes

	parent, err = x509.ParseCertificate(parentCert)
	require.NoError(t, err)
	return
}

func generateCerts(t *testing.T, opts generatedCertOpts) (rootCAs *x509.CertPool, certs [][]byte) {
	parentKey, parent, parentCert := generateCACert(t)

	// If we want the certificate to be expired by the time it's being verified.
	var expiry time.Duration
	if opts.expired {
		expiry = time.Microsecond
	} else {
		expiry = time.Hour
	}

	signingKey := parentKey
	if opts.useWrongKey {
		var err error
		signingKey, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)
	}

	signer, err := local.NewSigner(signingKey, parent, x509.ECDSAWithSHA256, &config.Signing{
		Default: &config.SigningProfile{
			Usage:    []string{"signing", "key encipherment", "server auth", "client auth"},
			Expiry:   expiry,
			Backdate: time.Minute,
			CSRWhitelist: &config.CSRWhitelist{
				PublicKey:          true,
				PublicKeyAlgorithm: true,
				SignatureAlgorithm: true,
			},
		},
	})
	require.NoError(t, err)

	csr := &cfcsr.CertificateRequest{
		KeyRequest: cfcsr.NewBasicKeyRequest(),
	}
	csrBytes, _, err := cfcsr.ParseRequest(csr)
	require.NoError(t, err)

	cn := opts.cn
	if cn == "" {
		cn = mtls.CentralCN.String()
	}

	leafCertPEM, err := signer.Sign(cfsigner.SignRequest{
		Request: string(csrBytes),
		Subject: &cfsigner.Subject{
			CN:           cn,
			SerialNumber: strconv.FormatInt(100, 10),
		},
	})

	decoded, _ := pem.Decode(leafCertPEM)
	require.NotNil(t, decoded)
	leafCert := decoded.Bytes

	rootCAs = x509.NewCertPool()
	if !opts.invalidRootCA {
		rootCAs.AddCert(parent)
	} else {
		_, fakeParent, _ := generateCACert(t)
		rootCAs.AddCert(fakeParent)
	}
	return rootCAs, [][]byte{leafCert, parentCert}
}

func TestVerifyPeerCertificateFunc(t *testing.T) {
	cases := []struct {
		name           string
		opts           generatedCertOpts
		errExpected    bool
		errMustContain string
	}{
		{
			"Happy Path",
			generatedCertOpts{},
			false,
			"",
		},
		{
			"Expired",
			generatedCertOpts{expired: true},
			true,
			"expired",
		},
		{
			"Bad CN",
			generatedCertOpts{cn: "INVALID"},
			true,
			"certificate is valid for INVALID, not",
		},
		{
			"Invalid Root CA",
			generatedCertOpts{invalidRootCA: true},
			true,
			"certificate signed by unknown authority",
		},
		{
			"Invalid Root CA",
			generatedCertOpts{useWrongKey: true},
			true,
			"ECDSA verification failure",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rootCAs, certs := generateCerts(t, c.opts)
			err := verifyPeerCertificateFunc(rootCAs)(certs, nil)
			if c.errExpected {
				assert.Error(t, err)
				if c.errMustContain != "" {
					assert.Contains(t, err.Error(), c.errMustContain)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
