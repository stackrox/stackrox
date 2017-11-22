package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"sync"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
)

var (
	/*
		CreateCertificate creates a new certificate based on a template.
		The following members of template are used:
			SerialNumber,
			Subject,
			NotBefore,
			NotAfter,
			KeyUsage,
			ExtKeyUsage,
			UnknownExtKeyUsage,
			BasicConstraintsValid,
			IsCA,
			MaxPathLen,
			SubjectKeyId,
			DNSNames,
			PermittedDNSDomainsCritical,
			PermittedDNSDomains,
			SignatureAlgorithm.
	*/
	tenYears     = time.Now().Add(10 * 365 * 24 * time.Hour)
	certTemplate = x509.Certificate{
		BasicConstraintsValid:       true,
		DNSNames:                    []string{"*.stackrox"},
		IsCA:                        true,
		KeyUsage:                    x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:                 []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		MaxPathLen:                  0,
		NotAfter:                    tenYears,
		NotBefore:                   time.Now().Add(-5 * time.Minute), //5 minute clock skew allowance
		PermittedDNSDomains:         nil,
		PermittedDNSDomainsCritical: false,
		SignatureAlgorithm:          x509.SHA256WithRSA,
		Subject: pkix.Name{
			Country:            []string{"US"},
			Organization:       []string{"StackRox"},
			OrganizationalUnit: []string{"StackRox"},
			Locality:           []string{"Mountain View"},
			CommonName:         "SSO SP Cert",
		},
	}
	certInit sync.Once

	log = logging.New("tls/keys")
)

// GenerateStackRoxKeyPair recreates a StackRox x509 Certificate and private key from scratch
func GenerateStackRoxKeyPair() (publicCert Certificate, privateKey PrivateKey, err error) {
	privateKey, err = generatePrivateKey()
	if err != nil {
		return
	}
	publicCert, err = generatePublicCert(privateKey)
	return

}

func generatePrivateKey() (privateKey PrivateKey, err error) {
	rsaKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return
	}
	derRSA := x509.MarshalPKCS1PrivateKey(rsaKey)
	return NewPrivateKey(string(derRSA))
}

func generatePublicCert(privateKey PrivateKey) (publicCert Certificate, err error) {
	rsaKey, err := x509.ParsePKCS1PrivateKey(privateKey.Bytes)
	if err != nil {
		return
	}
	publicKey := rsaKey.Public()
	//passing in identity certTemplate as parent and child creates a self-signed cert
	certDER, err := x509.CreateCertificate(rand.Reader, getCertTemplate(), getCertTemplate(), publicKey, rsaKey)
	if err != nil {
		return
	}
	return NewCertificate(string(certDER))
}

func getCertTemplate() *x509.Certificate {
	certInit.Do(func() {
		var err error
		certTemplate.SerialNumber, err = rand.Int(rand.Reader, big.NewInt(2<<20))
		if err != nil {
			log.Fatal(err)
		}
	})

	return &certTemplate
}
