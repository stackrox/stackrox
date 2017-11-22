package keys

import (
	"bytes"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

var (
	beginCertHeader      = []byte("-----BEGIN CERTIFICATE-----")
	endCertHeader        = []byte("-----END CERTIFICATE-----")
	errUnknownCertFormat = fmt.Errorf("Unable to determine format of provided certificate")
	errEmptyCert         = fmt.Errorf("Provided certificate is empty")
)

// Certificate contains a PEM formatted x509 certificate
type Certificate pem.Block

// NewCertificate encapsulates base64 DER cert, base64 PEM, and raw PEM encoded certs into a standard Certificate type
func NewCertificate(raw string) (cert Certificate, err error) {
	if len(raw) == 0 {
		err = errEmptyCert
		return
	}
	cert = Certificate(pem.Block{Type: "CERTIFICATE", Bytes: []byte(raw)})
	if _, err = cert.ToX509(); err == nil {
		return
	}
	if isPEM, bytes := cert.Key().fromPEM(); isPEM {
		return newCertFromPEM(bytes)
	}
	if isBase64, bytes := cert.Key().fromBase64(); isBase64 {
		return newCertFromBase64(bytes)
	}
	err = errUnknownCertFormat
	return
}

func newCertFromPEM(unknown []byte) (cert Certificate, err error) {
	if unknown == nil || len(unknown) == 0 {
		err = errEmptyCert
		return
	}
	cert = Certificate(pem.Block{Type: "CERTIFICATE", Bytes: unknown})
	if _, err = cert.ToX509(); err == nil {
		return
	}
	err = errUnknownCertFormat
	return
}

func newCertFromBase64(unknown []byte) (cert Certificate, err error) {
	if unknown == nil || len(unknown) == 0 {
		err = errEmptyCert
		return
	}
	cert = Certificate(pem.Block{Type: "CERTIFICATE", Bytes: unknown})
	if _, err = cert.ToX509(); err == nil {
		return
	}
	if isPEM, bytes := cert.Key().fromPEM(); isPEM {
		return newCertFromPEM(bytes)
	}
	err = errUnknownCertFormat
	return
}

// Key generalizes this Certificate into the common Key type, enabling public or private agnostic actions
func (c Certificate) Key() Key {
	return Key{
		Block:   pem.Block(c),
		keyType: Public,
	}
}

// ToX509 converts this Certificate into the golang X509 certificate type
func (c Certificate) ToX509() (*x509.Certificate, error) {
	return x509.ParseCertificate(c.Bytes)
}

// PublicKey retrieves the Public Key embedded within this Certificate
func (c Certificate) PublicKey() (key crypto.PublicKey, err error) {
	cert, err := c.ToX509()
	if err != nil {
		return
	}
	key = cert.PublicKey
	return
}

// reconstruct a PEM formatted cert, ignoring changes to whitespace and headers
// note: we may receive certs in a variety of formats, particularly when passed as JSON.
// While technically malformed, the user will interpret the lack of parsing as our problem, not theirs.
func reconstructPEM(raw []byte) (reconstructed []byte) {
	//remove headers
	noBeginHeader := bytes.Replace([]byte(raw), beginCertHeader, []byte(""), -1)
	noHeaders := bytes.Replace([]byte(noBeginHeader), endCertHeader, []byte(""), -1)
	//remove whitespace
	compressed := bytes.Join(bytes.Fields(noHeaders), []byte(""))
	//add headers again
	reconstructed = bytes.Join([][]byte{beginCertHeader, compressed, endCertHeader}, []byte("\n"))
	return
}
