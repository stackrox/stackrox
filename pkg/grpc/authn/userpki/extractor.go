package userpki

import (
	"context"
	"fmt"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

// NewExtractor returns an IdentityExtractor that will map identities based
// on certificates available in the ProviderContainer
func NewExtractor(manager ProviderContainer) authn.IdentityExtractor {
	return extractor{
		manager: manager,
	}
}

// ProviderContainer is an interface that ClientCAManager implements
type ProviderContainer interface {
	GetProviderForFingerprint(fingerprint string) authproviders.Provider
}

type extractor struct {
	manager ProviderContainer
}

func (i extractor) IdentityForRequest(ctx context.Context, ri requestinfo.RequestInfo) (authn.Identity, error) {
	// this auth identity provider is only relevant for API usage outside of the browser app. Inside the browser app,
	// tokens are used (with validation to ensure continuity of access). So we ignore certs if the authorization
	// header is set.
	authHeaders := ri.Metadata.Get("authorization")
	if len(authHeaders) > 0 {
		return nil, nil
	}

	if len(ri.VerifiedChains) != 1 {
		return nil, nil
	}
	log.Debugf("Looking up TLS trust for user cert chain: %+v", ri.VerifiedChains[0])
	for _, info := range ri.VerifiedChains[0] {
		provider := i.manager.GetProviderForFingerprint(info.CertFingerprint)
		if provider == nil {
			continue
		}
		userCert := ri.VerifiedChains[0][0]
		attributes := ExtractAttributes(userCert)
		identity := &identity{userCert, provider.ID(), nil, attributes}
		ud := &permissions.UserDescriptor{
			UserID:     identity.UID(),
			Attributes: attributes,
		}
		role, err := provider.RoleMapper().FromUserDescriptor(ctx, ud)
		if err != nil {
			return nil, err
		}
		identity.role = role
		return identity, nil
	}
	return nil, nil
}

// ExtractAttributes converts a subset of CertInfo into an attribute map for authorization
func ExtractAttributes(userCert requestinfo.CertInfo) map[string][]string {
	// TODO(ROX-2190)
	output := make(map[string][]string)
	output["CN"] = []string{userCert.Subject.CommonName}
	return output
}

type identity struct {
	info       requestinfo.CertInfo
	providerID string
	role       *storage.Role
	attributes map[string][]string
}

func (i *identity) Attributes() map[string][]string {
	return i.attributes
}

func (i *identity) FriendlyName() string {
	return i.info.Subject.CommonName
}

func (i *identity) User() *storage.UserInfo {
	return &storage.UserInfo{
		FriendlyName: i.info.Subject.CommonName,
		Role:         i.role,
	}
}

func (i *identity) Role() *storage.Role {
	return i.role
}

func (i *identity) Service() *storage.ServiceIdentity {
	return nil
}

func (i *identity) Expiry() time.Time {
	return i.info.NotAfter
}

func (i *identity) ExternalAuthProvider() authproviders.Provider {
	return nil
}

func (i *identity) UID() string {
	return fmt.Sprintf("pki/%s/%s", i.providerID, i.info.SerialNumber)
}
