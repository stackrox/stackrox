package cryptoutils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
)

// DerefPrivateKey dereferences a private key if it is a pointer internally. This is necessary as private keys are
// typically stored as pointers in Go TLS libraries (*ecdsa.PrivateKey, *rsa.PrivateKey), but `CreateSignature` from
// CT TLS expects a non-pointer object.
func DerefPrivateKey(pk crypto.PrivateKey) crypto.PrivateKey {
	switch k := pk.(type) {
	case *ecdsa.PrivateKey:
		return *k
	case *rsa.PrivateKey:
		return *k
	default:
		return k
	}
}
