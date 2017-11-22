package keys

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
)

// ImportKeyRequest provides a JSON template for submitting public and private keys
type ImportKeyRequest struct {
	PublicCert string `json:"public_cert" validate:"required,certificate"`
	PrivateKey string `json:"private_key" validate:"required,privatekey"`
}

// KeyType tracks if this key is public or private
type KeyType int

const (
	// Public indicates this is a public key
	Public KeyType = iota
	// Private indicates this is a private key
	Private
)

// Key implements functions common to both Public and Private keys
type Key struct {
	pem.Block
	keyType KeyType
}

// String implements the Stringer interface, providing a textual representation of this Certificate
func (k Key) String() string {
	return string(k.PEM())
}

// PEM returns this key in PEM format
func (k Key) PEM() []byte {
	return pem.EncodeToMemory(&k.Block)
}

// Base64Encode is a convenience function to encode certs in a JSON friendly format (base64)
// Creating a new Certificate from a base64 encoded cert will automatically decode it
func (k Key) Base64Encode() string {
	return base64.StdEncoding.EncodeToString(k.Bytes)
}

// Validate verifies that this key can be parsed
func (k Key) Validate() bool {
	var err error
	switch k.keyType {
	case Public:
		_, err = NewCertificate(string(k.Bytes))
	case Private:
		_, err = NewPrivateKey(string(k.Bytes))
	default:
		return false
	}
	return err == nil
}

// Base64 checks if this key is Base64 encoded and returns the decoded bytes
func (k Key) fromBase64() (isBase64 bool, decoded []byte) {
	decoded, err := base64.StdEncoding.DecodeString(string(k.Bytes))
	isBase64 = err == nil
	return
}

// PEM checks if this key is a valid PEM format and returns a pem block
func (k Key) fromPEM() (isPEM bool, raw []byte) {
	block, _ := pem.Decode(k.Bytes)
	isPEM = block != nil
	if isPEM {
		raw = block.Bytes
		return
	}
	reconstructed := reconstructPEM(k.Bytes)
	block, _ = pem.Decode(reconstructed)
	if block == nil {
		return false, raw
	}
	_, err := x509.ParseCertificate(block.Bytes)
	isPEM = err == nil
	if isPEM {
		raw = block.Bytes
		return
	}
	return
}
