package servicecerttoken

import (
	"encoding/base64"

	"github.com/google/certificate-transparency-go/tls"
)

const (
	// hashAlgo is the cryptographic hash algorithm to use for signatures.
	hashAlgo = tls.SHA256
	// TokenType is the prefix in the authorization header to identify the token type.
	TokenType = "ServiceCert"
)

var (
	// b64Enc is the base64 encoding flavor to use for tokens.
	b64Enc = base64.RawStdEncoding
)
