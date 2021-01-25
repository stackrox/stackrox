package mtls

import (
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"strings"
	"time"

	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	hostnameOverrides = map[storage.ServiceType]string{
		storage.ServiceType_ADMISSION_CONTROL_SERVICE: "admission-control",
	}
)

// Identity identifies a particular certificate.
type Identity struct {
	Subject Subject
	Serial  *big.Int
	Expiry  time.Time
}

// IdentityFromCert returns an mTLS (mutual TLS) identity for the given certificate.
func IdentityFromCert(cert requestinfo.CertInfo) Identity {
	return Identity{
		Subject: convertCertSubject(cert.Subject),
		Serial:  cert.SerialNumber,
		Expiry:  cert.NotAfter,
	}
}

// V1 returns the identity represented as a v1 API ServiceIdentity.
func (id Identity) V1() *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		Srl: &storage.ServiceIdentity_SerialStr{
			SerialStr: id.Serial.String(),
		},
		Type:         id.Subject.ServiceType,
		Id:           id.Subject.Identifier,
		InitBundleId: id.Subject.InitBundleID,
	}
}

// Subject encodes the parts of a certificate's identity.
type Subject struct {
	ServiceType  storage.ServiceType
	Identifier   string
	InitBundleID string
}

// CertificateOptions define options which are available at cert generation
type CertificateOptions struct {
	SerialNumber *big.Int
}

// NewSubject returns a new subject from the passed ID and service type
func NewSubject(id string, serviceType storage.ServiceType) Subject {
	return Subject{
		Identifier:  id,
		ServiceType: serviceType,
	}
}

// NewInitSubject returns a new subject from the passed ID and service type
func NewInitSubject(id string, serviceType storage.ServiceType, initBundleID uuid.UUID) Subject {
	return Subject{
		Identifier:   id,
		ServiceType:  serviceType,
		InitBundleID: initBundleID.String(),
	}
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

// AllHostnames returns all of the hostnames: e.g. central.stackrox.svc
func (s Subject) AllHostnames() []string {
	// Admission Controllers require the .svc suffix
	hostnames := []string{s.Hostname(), fmt.Sprintf("%s.svc", s.Hostname())}
	if s.ServiceType == storage.ServiceType_SENSOR_SERVICE {
		hostnames = append(hostnames, fmt.Sprintf("%s-webhook.stackrox.svc", hostname(s.ServiceType)))
	}
	return hostnames
}

func hostname(t storage.ServiceType) string {
	if hn := hostnameOverrides[t]; hn != "" {
		return hn
	}
	return strings.ToLower(strings.Split(t.String(), "_")[0])
}

// OU returns the Organizational Unit for the Subject.
func (s Subject) OU() string {
	return s.ServiceType.String()
}

// O returns the Organization for the Subject.
func (s Subject) O() string {
	return s.InitBundleID
}

// Name generates a cfssl Name for the subject, as a convenience.
func (s Subject) Name() cfcsr.Name {
	return cfcsr.Name{
		OU: s.OU(),
		O:  s.O(),
	}
}

func convertCertSubject(subject pkix.Name) Subject {
	s := SubjectFromCommonName(subject.CommonName)

	if len(subject.Organization) != 0 && subject.Organization[0] != "" {
		s.InitBundleID = uuid.FromStringOrNil(subject.Organization[0]).String()
	}
	return s
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
