package m2m

import (
	"context"
	"time"

	"github.com/pkg/errors"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
)

var (
	_ TokenExchanger = (*machineToMachineTokenExchanger)(nil)
)

// TokenExchanger will exchange a raw ID token to a Rox token (i.e. a Central access token).
// This will be done based on an auth machine to machine config.
//
//go:generate mockgen-wrapper
type TokenExchanger interface {
	ExchangeToken(ctx context.Context, rawIDToken string) (string, error)
	Provider() authproviders.Provider
	Config() *storage.AuthMachineToMachineConfig
}

type machineToMachineTokenExchanger struct {
	config            *storage.AuthMachineToMachineConfig
	verifier          tokenVerifier
	provider          authproviders.Provider
	issuer            tokens.Issuer
	roleDS            roleDataStore.DataStore
	roxClaimExtractor claimExtractor
}

// newTokenExchanger creates a new token exchanger based on an auth machine to machine config.
func newTokenExchanger(ctx context.Context, config *storage.AuthMachineToMachineConfig,
	roleDS roleDataStore.DataStore, issuerFactory tokens.IssuerFactory) (TokenExchanger, error) {
	tokenTTL, err := getTTLFromConfigs([]*storage.AuthMachineToMachineConfig{config})
	// Technically, this shouldn't happen, as the config is expected to be validated beforehand (i.e. when added to the
	// data store).
	if err != nil {
		return nil, errors.Wrap(err, "parsing token expiration duration")
	}

	verifier, err := tokenVerifierFromConfigData(ctx, config.GetType(), config.GetIssuer(), config.GetAudience())
	if err != nil {
		return nil, err
	}
	claimToRoleMapper := newRoleMapperFromMappings(config.GetMappings(), roleDS)
	provider := newProvider(config.GetIssuer(), config.GetType().String(), claimToRoleMapper)
	roxClaimExtractor := newClaimExtractorForConfigType(config.GetType())
	issuer, err := issuerFactory.CreateIssuer(provider, tokens.WithTTL(tokenTTL))
	if err != nil {
		return nil, errors.Wrap(err, "creating token issuer")
	}

	return &machineToMachineTokenExchanger{
		config:            config,
		issuer:            issuer,
		verifier:          verifier,
		provider:          provider,
		roxClaimExtractor: roxClaimExtractor,
		roleDS:            roleDS,
	}, nil
}

func getTTLFromConfigs(configs []*storage.AuthMachineToMachineConfig) (time.Duration, error) {
	if len(configs) == 0 {
		return time.Duration(0), errox.InvalidArgs.New("should have at least one config in input")
	}
	ttl, err := time.ParseDuration(configs[0].GetTokenExpirationDuration())
	if err != nil {
		return time.Duration(0), errox.InvalidArgs.CausedBy(err)
	}
	for _, config := range configs[1:] {
		configTTL, err := time.ParseDuration(config.GetTokenExpirationDuration())
		if err != nil {
			return time.Duration(0), errox.InvalidArgs.CausedBy(err)
		}
		if configTTL < ttl {
			ttl = configTTL
		}
	}
	return ttl, nil
}

func (m *machineToMachineTokenExchanger) Provider() authproviders.Provider {
	return m.provider
}

func (m *machineToMachineTokenExchanger) ExchangeToken(ctx context.Context, rawIDToken string) (string, error) {
	idToken, err := m.verifier.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return "", errox.NoCredentials.New("ID token is invalid").CausedBy(err)
	}

	log.Debugf("Successfully validated ID token (sub=%q) for issuer %q", idToken.Subject, m.config.GetIssuer())

	claims, err := m.roxClaimExtractor.ExtractClaims(idToken)
	if err != nil {
		return "", errox.NoCredentials.New("extracting claims from ID token").CausedBy(err)
	}

	log.Debugf("String claims of the ID token (sub=%q): %+v", idToken.Subject, claims)

	// We attempt to resolve the roles for the claims here. In case no roles are given for the ID token, we reject it.
	// Additionally, since the context will have no access, elevate it locally to fetch roles.
	resolveRolesCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))
	_, err = resolveRolesForClaims(resolveRolesCtx, claims, m.roleDS, parseMappings(m.config.GetMappings()))
	if err != nil {
		return "", errox.NoCredentials.New("resolving roles for id token").CausedBy(err)
	}

	roxClaims, err := m.roxClaimExtractor.ExtractRoxClaims(claims)
	if err != nil {
		return "", errox.NoCredentials.New("creating claims for Central token").CausedBy(err)
	}

	log.Debugf("Rox claims for the ID token (sub=%q): %+v", idToken.Subject, roxClaims)

	tokenInfo, err := m.issuer.Issue(ctx, roxClaims)
	if err != nil {
		return "", errox.NoCredentials.New("issuing Central token").CausedBy(err)
	}

	return tokenInfo.Token, nil
}

func (m *machineToMachineTokenExchanger) Config() *storage.AuthMachineToMachineConfig {
	return m.config
}
