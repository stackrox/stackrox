package mtls

import (
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"strings"
	"time"

	cfcsr "github.com/cloudflare/cfssl/csr"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// Identity identifies a particular certificate.
type Identity struct {
	Subject   Subject
	Serial    *big.Int
	NotBefore time.Time
	Expiry    time.Time
}

// IdentityFromCert returns an mTLS (mutual TLS) identity for the given certificate.
func IdentityFromCert(cert CertInfo) Identity {
	return Identity{
		Subject:   convertCertSubject(cert.Subject),
		Serial:    cert.SerialNumber,
		NotBefore: cert.NotBefore,
		Expiry:    cert.NotAfter,
	}
}

// V1 returns the identity represented as a v1 API ServiceIdentity.
func (id Identity) V1() *storage.ServiceIdentity {
	return &storage.ServiceIdentity{
		SerialStr:    id.Serial.String(),
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
	TenantID     string
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
func NewInitSubject(id string, serviceType storage.ServiceType, initBundleID uuid.UUID, tenantID string) Subject {
	return Subject{
		Identifier:   id,
		ServiceType:  serviceType,
		InitBundleID: initBundleID.String(),
		TenantID:     tenantID,
	}
}

// CN returns the Common Name that represents this subject's identity.
func (s Subject) CN() string {
	return fmt.Sprintf("%s: %s", s.ServiceType, s.Identifier)
}

// Hostname returns the hostname that should represent this subject
// as a Subject Alternative Name in the default case (stackrox namespace).
func (s Subject) Hostname() string {
	return s.HostnameForNamespace("stackrox")
}

// HostnameForNamespace returns the hostname that should represent this subject
// in an arbitrary namespace.
func (s Subject) HostnameForNamespace(namespace string) string {
	return fmt.Sprintf("%s.%s", hostname(s.ServiceType), namespace)
}

// AllHostnames returns all of the hostnames for the default case: e.g. central.stackrox.svc
func (s Subject) AllHostnames() []string {
	return s.AllHostnamesForNamespace("stackrox")
}

// AllHostnamesForNamespace returns all of the hostnames for a specific namespace: e.g. central.my-namespace.svc
func (s Subject) AllHostnamesForNamespace(namespace string) []string {
	// Admission Controllers require the .svc suffix
	hostnames := []string{s.HostnameForNamespace(namespace), fmt.Sprintf("%s.svc", s.HostnameForNamespace(namespace))}
	if s.ServiceType == storage.ServiceType_SENSOR_SERVICE {
		hostnames = append(hostnames, fmt.Sprintf("%s-webhook.%s.svc", hostname(s.ServiceType), namespace))
	}
	return hostnames
}

func hostname(t storage.ServiceType) string {
	lastIdx := strings.LastIndex(t.String(), "_")
	if lastIdx == -1 {
		return t.String()
	}
	val := t.String()[:lastIdx]
	val = strings.ReplaceAll(val, "_", "-")
	return strings.ToLower(val)
}

// OU returns the Organizational Unit for the Subject.
func (s Subject) OU() string {
	return s.ServiceType.String()
}

// O returns the Organization for the Subject.
func (s Subject) O() string {
	return s.InitBundleID
}

// State returns the State for the Subject.
func (s Subject) State() string {
	return s.TenantID
}

// Name generates a cfssl Name for the subject, as a convenience.
func (s Subject) Name() cfcsr.Name {
	return cfcsr.Name{
		OU: s.OU(),
		O:  s.O(),
		ST: s.State(),
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
