package service

import (
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func TestWrapMTLSIdentity(t *testing.T) {
	for name, tc := range map[string]struct {
		id mtls.Identity
	}{
		"service identity": {
			id: mtls.Identity{
				Subject:   mtls.NewSubject("test-sensor", storage.ServiceType_SENSOR_SERVICE),
				Serial:    big.NewInt(42),
				NotBefore: time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
				Expiry:    time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		"service identity with init bundle": {
			id: mtls.Identity{
				Subject:   mtls.NewInitSubject("test-sensor", storage.ServiceType_SENSOR_SERVICE, uuid.FromStringOrPanic("01928374-0192-0192-0192-019283746564")),
				Serial:    big.NewInt(43),
				NotBefore: time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
				Expiry:    time.Date(2050, time.February, 1, 0, 0, 0, 0, time.UTC),
			},
		},
	} {
		t.Run(name, func(it *testing.T) {
			id := WrapMTLSIdentity(tc.id)
			var _ authn.Identity = id

			assert.Equal(it, fmt.Sprintf("mtls:%s@%v", tc.id.Subject.Identifier, tc.id.Serial), id.UID())
			assert.Equal(it, tc.id.Subject.CN(), id.FriendlyName())
			assert.Equal(it, tc.id.Subject.CN(), id.FullName())
			// Service identities have no user-facing properties.
			assert.Nil(it, id.User())
			assert.Nil(it, id.Permissions())
			assert.Nil(it, id.Roles())
			assert.Nil(it, id.Attributes())
			assert.Nil(it, id.ExternalAuthProvider())
			protoassert.Equal(it, tc.id.V1(), id.Service())

			validFrom, validUntil := id.ValidityPeriod()
			assert.Equal(it, tc.id.NotBefore, validFrom)
			assert.Equal(it, tc.id.Expiry, validUntil)
		})
	}
}

// TestWrapMTLSIdentityFromCertInfo ensures that the mTLS identity fields are
// faithfully derived from a certificate info, as done for verified chains.
func TestWrapMTLSIdentityFromCertInfo(t *testing.T) {
	certInfo := mtls.CertInfo{
		Subject:         pkix.Name{CommonName: "SENSOR_SERVICE: test-sensor"},
		NotBefore:       time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		NotAfter:        time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC),
		SerialNumber:    big.NewInt(42),
		CertFingerprint: "test-fingerprint",
	}

	id := WrapMTLSIdentity(mtls.IdentityFromCert(certInfo))

	assert.Equal(t, "mtls:test-sensor@42", id.UID())
	assert.Equal(t, "SENSOR_SERVICE: test-sensor", id.FriendlyName())
	protoassert.Equal(t, &storage.ServiceIdentity{
		SerialStr: "42",
		Type:      storage.ServiceType_SENSOR_SERVICE,
		Id:        "test-sensor",
	}, id.Service())
}
