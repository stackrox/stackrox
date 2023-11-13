package m197tom198

import (
	"context"

	"github.com/pkg/errors"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/store/authproviders"
	"github.com/stackrox/rox/migrator/migrations/m_197_to_m_198_add_oidc_claim_mappings/store/groups"
	"github.com/stackrox/rox/migrator/types"
	"github.com/stackrox/rox/pkg/auth/authproviders/oidc"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
)

var (
	log       = logging.LoggerForModule()
	batchSize = 500
)

const (
	orgIDKey                      = "orgid"
	accountIDClaim                = "account_id"
	groupsKey                     = "groups"
	isOrgAdminKey                 = "is_org_admin"
	orgAdminGroupsValue           = "org_admin"
	declarativeProviderLogMessage = "Declarative auth provider with id %s and name %s uses claims that were removed from the list of default claims. " +
		"Please add claim mapping for account_id or is_org_admin if you are using any of these claims"
)

func migrate(database *types.Databases) error {
	ctx := sac.WithAllAccess(context.Background())
	groupStore := groups.New(database.PostgresDB)
	authProviderStore := authproviders.New(database.PostgresDB)
	oidcAuthProviders := make([]*storage.AuthProvider, 0, batchSize)

	err := authProviderStore.Walk(ctx, func(obj *storage.AuthProvider) error {
		if obj.GetType() == oidc.TypeName {
			oidcAuthProviders = append(oidcAuthProviders, obj)
			if len(oidcAuthProviders) == batchSize {
				if err := migrateAuthProviderClaims(ctx, groupStore, authProviderStore, oidcAuthProviders); err != nil {
					return err
				}
				oidcAuthProviders = oidcAuthProviders[:0]
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return migrateAuthProviderClaims(ctx, groupStore, authProviderStore, oidcAuthProviders)
}

func migrateAuthProviderClaims(ctx context.Context, groupStore groups.Store, providerStore authproviders.Store, providers []*storage.AuthProvider) error {
	providerToGroups := make(map[string][]*storage.Group, len(providers))
	for _, provider := range providers {
		providerToGroups[provider.GetId()] = make([]*storage.Group, 0)
	}
	err := groupStore.Walk(ctx, func(obj *storage.Group) error {
		authProviderID := obj.GetProps().GetAuthProviderId()
		if _, ok := providerToGroups[authProviderID]; ok {
			providerToGroups[authProviderID] = append(providerToGroups[authProviderID], obj)
		}
		return nil
	})
	if err != nil {
		return errors.Wrap(err, "failed to get groups from the store")
	}
	migratedProviders := make([]*storage.AuthProvider, 0)
	migratedGroups := make([]*storage.Group, 0)
	for _, provider := range providers {
		providerGroups := providerToGroups[provider.GetId()]
		fixOrgID := (hasRequiredAttributeKey(orgIDKey, provider) || hasGroupKey(orgIDKey, providerGroups)) && !hasClaimMapping(accountIDClaim, orgIDKey, provider.GetClaimMappings())
		if fixOrgID {
			setClaimMapping(provider, accountIDClaim, orgIDKey)
		}
		var fixIsOrgAdmin bool
		var groupToMigrate *storage.Group
		for _, requiredAttribute := range provider.GetRequiredAttributes() {
			if requiredAttribute.GetAttributeKey() == groupsKey && requiredAttribute.GetAttributeValue() == orgAdminGroupsValue {
				setClaimMapping(provider, isOrgAdminKey, orgAdminGroupsValue)
				requiredAttribute.AttributeKey = orgAdminGroupsValue
				requiredAttribute.AttributeValue = "true"
				fixIsOrgAdmin = true
			}
		}
		for _, group := range providerGroups {
			if group.GetProps().GetKey() == groupsKey && group.GetProps().GetValue() == orgAdminGroupsValue {
				setClaimMapping(provider, isOrgAdminKey, orgAdminGroupsValue)
				groupToMigrate = group
				groupToMigrate.Props.Key = orgAdminGroupsValue
				groupToMigrate.Props.Value = "true"
				fixIsOrgAdmin = true
				break
			}
		}

		if fixOrgID || fixIsOrgAdmin {
			if provider.GetTraits().GetOrigin() != storage.Traits_IMPERATIVE {
				log.Errorf(declarativeProviderLogMessage, provider.GetId(), provider.GetName())
			} else {
				log.Warnf("Auth provider with id %s and name %s uses claims org_id, org_admin that were removed from the list of default claims. "+
					"Claims were automatically added to claim mappings, groups were modified accordingly.", provider.GetId(), provider.GetName())
				migratedProviders = append(migratedProviders, provider)
				if groupToMigrate != nil {
					migratedGroups = append(migratedGroups, groupToMigrate)
				}
			}
		}
	}
	if err := groupStore.UpsertMany(ctx, migratedGroups); err != nil {
		return errors.Wrap(err, "failed to upsert migrated groups")
	}
	if err := providerStore.UpsertMany(ctx, migratedProviders); err != nil {
		return errors.Wrap(err, "failed to upsert migrated providers")
	}
	return nil
}

func setClaimMapping(provider *storage.AuthProvider, claim string, key string) {
	if provider.GetClaimMappings() == nil {
		provider.ClaimMappings = map[string]string{}
	}
	provider.ClaimMappings[claim] = key
}

func hasClaimMapping(expectedPath, expectedMapping string, mappings map[string]string) bool {
	mapping, ok := mappings[expectedPath]
	return ok && mapping == expectedMapping
}

func hasGroupKey(key string, groups []*storage.Group) bool {
	for _, group := range groups {
		if group.GetProps().GetKey() == key {
			return true
		}
	}
	return false
}

func hasRequiredAttributeKey(attributeKey string, provider *storage.AuthProvider) bool {
	for _, requiredAttribute := range provider.GetRequiredAttributes() {
		if requiredAttribute.GetAttributeKey() == attributeKey {
			return true
		}
	}
	return false
}
