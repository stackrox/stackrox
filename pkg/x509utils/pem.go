package x509utils

import (
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// ParseCertificatePEM parses a single PEM-encoded X.509 certificate.
func ParseCertificatePEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}
	return x509.ParseCertificate(block.Bytes)
}

// ParseCertificatesPEM parses all PEM-encoded certificates from the given data.
func ParseCertificatesPEM(data []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	for len(data) > 0 {
		var block *pem.Block
		block, data = pem.Decode(data)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			continue
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		certs = append(certs, cert)
	}
	if len(certs) == 0 {
		return nil, errors.New("no certificates found in PEM data")
	}
	return certs, nil
}

// EncodeCertificatesPEM encodes a slice of certificates as PEM.
func EncodeCertificatesPEM(certs []*x509.Certificate) []byte {
	var buf []byte
	for _, cert := range certs {
		buf = append(buf, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})...)
	}
	return buf
}

// PEMToCertPool parses PEM-encoded certificates and returns a CertPool.
func PEMToCertPool(data []byte) (*x509.CertPool, error) {
	certs, err := ParseCertificatesPEM(data)
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	for _, cert := range certs {
		pool.AddCert(cert)
	}
	return pool, nil
}

// ParsePrivateKeyPEM parses a PEM-encoded private key (PKCS#8, PKCS#1, or EC).
func ParsePrivateKeyPEM(data []byte) (crypto.Signer, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			key, err = x509.ParseECPrivateKey(block.Bytes)
			if err != nil {
				return nil, errors.New("failed to parse private key")
			}
		}
	}

	signer, ok := key.(crypto.Signer)
	if !ok {
		return nil, errors.New("private key does not implement crypto.Signer")
	}
	return signer, nil
}
