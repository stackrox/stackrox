package userpki

import (
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/mtls"
)

const (
	userFingerprint = "test-user-fingerprint"
	caFingerprint   = "test-user-ca-fingerprint"
)

var (
	errDummy = errors.New("dummy test error")

	testNotAfter = time.Date(2050, time.January, 1, 0, 0, 0, 0, time.UTC)

	testRolePermissions = map[string]storage.Access{
		"Deployment": storage.Access_READ_ACCESS,
	}

	userCert = mtls.CertInfo{
		Subject: pkix.Name{
			CommonName:         "UserA",
			Organization:       []string{"Stackrox"},
			OrganizationalUnit: []string{"GroupA"},
			Country:            []string{"US"},
		},
		EmailAddresses:  []string{"usera@stackrox.io"},
		SerialNumber:    big.NewInt(1),
		NotAfter:        testNotAfter,
		CertFingerprint: userFingerprint,
	}

	caCert = mtls.CertInfo{
		Subject:         pkix.Name{CommonName: "User CA"},
		CertFingerprint: caFingerprint,
	}

	expectedUserAttributes = map[string][]string{
		"userid": {"userpki:" + userFingerprint},
		"name":   {"UserA"},
		"email":  {"usera@stackrox.io"},
		"groups": {"GroupA"},
		"CN":     {"UserA"},
		"C":      {"US"},
		"O":      {"Stackrox"},
		"OU":     {"GroupA"},
		"DN":     {"CN=UserA,OU=GroupA,O=Stackrox,C=US"},
	}
)

// fakeProviderContainer is a test stand-in for the ProviderContainer interface.
type fakeProviderContainer struct {
	providers map[string]authproviders.Provider
}

func (c *fakeProviderContainer) GetProviderForFingerprint(fingerprint string) authproviders.Provider {
	return c.providers[fingerprint]
}
