package mtls

import (
	"crypto/x509/pkix"
	"math/big"
	"time"
)

// CertInfo is the relevant (for us) fraction of a X.509 certificate that can safely be serialized.
type CertInfo struct {
	Subject             pkix.Name
	NotBefore, NotAfter time.Time
	EmailAddresses      []string
	SerialNumber        *big.Int
	CertFingerprint     string
}
