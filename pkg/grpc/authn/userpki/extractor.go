package userpki

import (
	"context"
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
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

	// We need all access for retrieving roles and upserting user info. Note that this context
	// is not propagated to the user, so the user itself does not get any escalated privileges.
	// Conversely, the context can't contain any access scope information because the identity has
	// not yet been extracted, so all code called with this context *must not* depend on a user
	// identity.
	ctx = sac.WithAllAccess(ctx)

	log.Debugf("Looking up TLS trust for user cert chain: %+v", ri.VerifiedChains[0])
	for _, info := range ri.VerifiedChains[0] {
		provider := i.manager.GetProviderForFingerprint(info.CertFingerprint)
		if provider == nil {
			continue
		}
		userCert := ri.VerifiedChains[0][0]
		attributes := ExtractAttributes(userCert)
		identity := &identity{
			info:       userCert,
			provider:   provider,
			attributes: attributes,
		}
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

type attributes map[string][]string

func (a attributes) add(key string, values ...string) {
	if len(values) == 0 {
		return
	}
	a[key] = append(a[key], values...)
}

// ExtractAttributes converts a subset of CertInfo into an attribute map for authorization
func ExtractAttributes(userCert requestinfo.CertInfo) map[string][]string {
	output := make(attributes)
	// these are the canonical stackrox attributes we use in the UI
	output.add("userid", userID(userCert))
	output.add("name", userCert.Subject.CommonName)
	output.add("email", userCert.EmailAddresses...)
	output.add("groups", userCert.Subject.OrganizationalUnit...)

	// standard LDAP-like attribute naming for external systems
	output["CN"] = output["name"]
	output.add("C", userCert.Subject.Country...)
	output.add("O", userCert.Subject.Organization...)
	output.add("OU", userCert.Subject.OrganizationalUnit...)
	output.add("L", userCert.Subject.Locality...)
	output.add("ST", userCert.Subject.Province...)
	output.add("STREET", userCert.Subject.StreetAddress...)
	output.add("POSTALCODE", userCert.Subject.PostalCode...)
	output.add("DN", userCert.Subject.String())

	return output
}

type identity struct {
	info       requestinfo.CertInfo
	provider   authproviders.Provider
	role       *storage.Role
	attributes map[string][]string
}

func (i *identity) Attributes() map[string][]string {
	return i.attributes
}

func (i *identity) FriendlyName() string {
	return i.info.Subject.CommonName
}

func (i *identity) FullName() string {
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
	return i.provider
}

func (i *identity) UID() string {
	return userID(i.info)
}

func userID(info requestinfo.CertInfo) string {
	return "userpki:" + info.CertFingerprint
}
