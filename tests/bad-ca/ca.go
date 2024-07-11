package bad_ca

import _ "embed"

// CustomCertPem is a certificate for untrusted-root.invalid domain that can be used for CA Setup Testing
//
//go:embed untrusted-root.invalid.crt
var CustomCertPem string

// SelfSignedCertPem is a certificate for self-signed.invalid domain that can be used for CA Setup Testing
//
//go:embed self-signed.invalid.crt
var SelfSignedCertPem string
