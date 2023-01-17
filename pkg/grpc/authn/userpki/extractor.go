package userpki

import (
	"context"
	"time"

	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/permissions"
	"github.com/stackrox/rox/pkg/grpc/authn"
	"github.com/stackrox/rox/pkg/grpc/requestinfo"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/mtls"
	"github.com/stackrox/rox/pkg/sac"
)

const (
	cacheSize          = 500
	rateLimitFrequency = 5 * time.Minute
	logBurstSize       = 5
)

var (
	log = logging.NewRateLimitLogger(logging.LoggerForModule(), cacheSize, 1, rateLimitFrequency, logBurstSize)
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

	if len(ri.VerifiedChains) == 0 {
		return nil, nil
	}

	// We need all access for retrieving roles and upserting user info. Note that this context
	// is not propagated to the user, so the user itself does not get any escalated privileges.
	// Conversely, the context can't contain any access scope information because the identity has
	// not yet been extracted, so all code called with this context *must not* depend on a user
	// identity.
	ctx = sac.WithAllAccess(ctx)

	for _, chain := range ri.VerifiedChains {
		log.Debugf("Looking up TLS trust for user cert chain: %+v", chain)
		for _, info := range chain {
			provider := i.manager.GetProviderForFingerprint(info.CertFingerprint)
			if provider == nil {
				continue
			}
			userCert := chain[0]
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
			resolvedRoles, err := provider.RoleMapper().FromUserDescriptor(ctx, ud)
			if err != nil {
				log.WarnL(ri.Hostname, "Token validation failed for hostname %v: %v", ri.Hostname, err)
				return nil, err
			}
			identity.resolvedRoles = resolvedRoles
			return identity, nil
		}
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
func ExtractAttributes(userCerts ...mtls.CertInfo) map[string][]string {
	output := make(attributes)

	for _, userCert := range userCerts {
		// these are the canonical stackrox attributes we use in the UI
		output.add(authproviders.UseridAttribute, userID(userCert))
		output.add(authproviders.NameAttribute, userCert.Subject.CommonName)
		output.add(authproviders.EmailAttribute, userCert.EmailAddresses...)
		output.add(authproviders.GroupsAttribute, userCert.Subject.OrganizationalUnit...)

		// standard LDAP-like attribute naming for external systems
		output["CN"] = output[authproviders.NameAttribute]
		output.add("C", userCert.Subject.Country...)
		output.add("O", userCert.Subject.Organization...)
		output.add("OU", userCert.Subject.OrganizationalUnit...)
		output.add("L", userCert.Subject.Locality...)
		output.add("ST", userCert.Subject.Province...)
		output.add("STREET", userCert.Subject.StreetAddress...)
		output.add("POSTALCODE", userCert.Subject.PostalCode...)
		output.add("DN", userCert.Subject.String())
	}

	return output
}
