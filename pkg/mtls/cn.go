package mtls

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
)

// Identity identifies a particular certificate.
type Identity struct {
	Subject Subject
	Serial  *big.Int
	Expiry  time.Time
}

// IdentityFromCert returns an mTLS identity for the given certificate.
func IdentityFromCert(cert requestinfo.CertInfo) Identity {
	return Identity{
		Subject: SubjectFromCommonName(cert.Subject.CommonName),
		Serial:  cert.SerialNumber,
		Expiry:  cert.NotAfter,
	}
}

// V1 returns the identity represented as a v1 API ServiceIdentity.
func (id Identity) V1() *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Serial: id.Serial.Int64(),
		Type:   id.Subject.ServiceType,
		Id:     id.Subject.Identifier,
	}
}

// Subject encodes the parts of a certificate's identity.
type Subject struct {
	ServiceType storage.ServiceType
	Identifier  string
}

// CN returns the Common Name that represents this subject's identity.
func (s Subject) CN() string {
	return fmt.Sprintf("%s: %s", s.ServiceType, s.Identifier)
}

// Hostname returns the hostname that should represent this subject
// as a Subject Alternative Name.
func (s Subject) Hostname() string {
	return fmt.Sprintf("%s.stackrox", hostname(s.ServiceType))
}

func hostname(t storage.ServiceType) string {
	return strings.ToLower(strings.Split(t.String(), "_")[0])
}

// OU returns the Organizational Unit for the Subject.
func (s Subject) OU() string {
	return s.ServiceType.String()
}

// Name generates a cfssl Name for the subject, as a convenience.
func (s Subject) Name() cfcsr.Name {
	return cfcsr.Name{
		OU: s.OU(),
	}
}

// SubjectFromCommonName parses a CN string into a Subject.
func SubjectFromCommonName(s string) Subject {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) == 2 {
		return Subject{
			ServiceType: storage.ServiceType(storage.ServiceType_value[parts[0]]),
			Identifier:  strings.TrimSpace(parts[1]),
		}
	}
	return Subject{
		Identifier: s,
	}
}
