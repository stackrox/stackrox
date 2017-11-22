package keys

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"reflect"
)

const unrecognizedKeyType = `Unrecognized Key Type`

var (
	errUnknownEncoding  = fmt.Errorf("Unable to determine encoding of provided private key")
	errUnknownKeyFormat = fmt.Errorf("Unable to determine format of provided private key")
	errEmptyKey         = fmt.Errorf("Provided private key is empty")
)

// PrivateKey is the raw ASN.1 DER bytes of an RSA Private Key
type PrivateKey pem.Block

// NewPrivateKey converts base64 DER private key, base64 PEM, and raw PEM encoded keys into a standard PrivateKey type
func NewPrivateKey(raw string) (priv PrivateKey, err error) {
	if len(raw) == 0 {
		err = errEmptyKey
		return
	}
	priv = PrivateKey(pem.Block{Type: "PRIVATE KEY", Bytes: []byte(raw)})
	if err = priv.recognizedFormat(); err == nil {
		return
	}
	if isPEM, bytes := priv.Key().fromPEM(); isPEM {
		return newPrivFromPEM(bytes)
	}
	if isBase64, bytes := priv.Key().fromBase64(); isBase64 {
		return newPrivKeyFromBase64(bytes)
	}
	err = errUnknownEncoding
	return
}

// SignatureKey returns a generalized golang private key type used for signing and verifying signatures
// This function abstracts the signature algorithm; it can be RSA or ECDSA.
func (p PrivateKey) SignatureKey() crypto.PrivateKey {
	return p.parse()
}

// Key generalizes this Certificate into the common Key type, enabling public or private agnostic actions
func (p PrivateKey) Key() Key {
	return Key{
		Block:   pem.Block(p),
		keyType: Private,
	}
}

func newPrivFromPEM(raw []byte) (priv PrivateKey, err error) {
	if raw == nil || len(raw) == 0 {
		err = errEmptyKey
		return
	}
	priv = PrivateKey(pem.Block{Type: "PRIVATE KEY", Bytes: raw})
	if err = priv.recognizedFormat(); err == nil {
		return
	}
	err = errUnknownKeyFormat
	return
}

func newPrivKeyFromBase64(raw []byte) (priv PrivateKey, err error) {
	if raw == nil || len(raw) == 0 {
		err = errEmptyKey
		return
	}
	priv = PrivateKey(pem.Block{Type: "PRIVATE KEY", Bytes: raw})
	if err = priv.recognizedFormat(); err == nil {
		return
	}
	if isPEM, bytes := priv.Key().fromPEM(); isPEM {
		return newPrivFromPEM(bytes)
	}
	err = errUnknownKeyFormat
	return
}

func (p PrivateKey) parse() (key crypto.PrivateKey) {
	if key, ecErr := x509.ParseECPrivateKey(p.Bytes); ecErr == nil {
		return key
	}
	if key, pkcs1Err := x509.ParsePKCS1PrivateKey(p.Bytes); pkcs1Err == nil {
		return key
	}
	if key, pkcs8Err := x509.ParsePKCS8PrivateKey(p.Bytes); pkcs8Err == nil {
		return key
	}
	return
}

func (p PrivateKey) implementation() string {
	switch t := p.parse().(type) {
	case *rsa.PrivateKey:
		return reflect.TypeOf(t).Name()
	case *ecdsa.PrivateKey:
		return reflect.TypeOf(t).Name()
	case crypto.PrivateKey:
		return reflect.TypeOf(t).Name()
	default:
		return unrecognizedKeyType
	}
}

func (p PrivateKey) recognizedFormat() (err error) {
	if p.implementation() == unrecognizedKeyType {
		err = errUnknownKeyFormat
	}
	return
}
