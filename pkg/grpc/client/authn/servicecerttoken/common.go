package servicecerttoken

import (
	"encoding/base64"

	"github.com/google/certificate-transparency-go/tls"
)

const (
	// HashAlgo is the cryptographic hash algorithm to use for signatures.
	HashAlgo = tls.SHA256
	// TokenType is the prefix in the authorization header to identify the token type.
	TokenType = "ServiceCert"
)

var (
	// TokenB64Enc is the base64 encoding flavor to use for tokens.
	TokenB64Enc = base64.RawStdEncoding
)
