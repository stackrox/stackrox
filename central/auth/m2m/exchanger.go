package m2m

import (
	"context"
	"slices"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/set"
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
	Configs() []*storage.AuthMachineToMachineConfig
	GetTokenExpirationDuration() time.Duration
	GetType() storage.AuthMachineToMachineConfig_Type
}

type machineToMachineTokenExchanger struct {
	exchangerType     storage.AuthMachineToMachineConfig_Type
	configs           []*storage.AuthMachineToMachineConfig
	configMappings    []*storage.AuthMachineToMachineConfig_Mapping
	verifiers         []tokenVerifier
	provider          authproviders.Provider
	issuer            tokens.Issuer
	roleDS            roleDataStore.DataStore
	roxClaimExtractor claimExtractor
	tokenTTL          time.Duration
	mappings          []parsedMapping
}

// newTokenExchanger creates a new token exchanger based on an auth machine to machine config.
func newTokenExchanger(
	ctx context.Context,
	config *storage.AuthMachineToMachineConfig,
	roleDS roleDataStore.DataStore,
	issuerFactory tokens.IssuerFactory,
) (TokenExchanger, error) {
	configs := []*storage.AuthMachineToMachineConfig{config}
	return newMultiConfigTokenExchanger(ctx, configs, roleDS, issuerFactory)
}

func newMultiConfigTokenExchanger(
	ctx context.Context,
	configs []*storage.AuthMachineToMachineConfig,
	roleDS roleDataStore.DataStore,
	issuerFactory tokens.IssuerFactory,
) (TokenExchanger, error) {
	if err := validateConfigConsistency(configs); err != nil {
		return nil, errors.Wrap(err, "creating token exchanger")
	}
	// The step above should ensure that there is at least one config in input,
	// and that all configs share the same issuer and type.
	exchangerType := configs[0].GetType()
	exchangerIssuer := configs[0].GetIssuer()

	tokenTTL, err := getTTLFromConfigs(configs)
	// Technically, this shouldn't happen, as the config is expected to be validated beforehand (i.e. when added to the
	// data store).
	if err != nil {
		return nil, errors.Wrap(err, "parsing token expiration duration")
	}

	audiences := set.NewStringSet()
	mappings := make([]*storage.AuthMachineToMachineConfig_Mapping, 0)
	parsedMappings := make([]parsedMapping, 0)
	for _, cfg := range configs {
		audiences.Add(cfg.GetAudience())
		mappings = append(mappings, cfg.GetMappings()...)
		parsedMappings = append(
			parsedMappings,
			parseMappings(cfg.GetMappings(), cfg.GetAudience())...,
		)
	}
	verifiers := make([]tokenVerifier, 0, audiences.Cardinality())
	for audience := range audiences {
		v, err := tokenVerifierFromConfigData(ctx, exchangerType, exchangerIssuer, audience)
		if err != nil {
			return nil, err
		}
		verifiers = append(verifiers, v)
	}
	claimToRoleMapper := newRoleMapperFromParsedMappings(parsedMappings, roleDS)
	provider := newProvider(exchangerIssuer, exchangerType.String(), claimToRoleMapper)
	roxClaimExtractor := newClaimExtractorForConfigType(exchangerType)
	issuer, err := issuerFactory.CreateIssuer(provider, tokens.WithTTL(tokenTTL))
	if err != nil {
		return nil, errors.Wrap(err, "creating token issuer")
	}

	return &machineToMachineTokenExchanger{
		exchangerType:     exchangerType,
		configs:           configs,
		issuer:            issuer,
		verifiers:         verifiers,
		provider:          provider,
		roxClaimExtractor: roxClaimExtractor,
		roleDS:            roleDS,
		tokenTTL:          tokenTTL,
		mappings:          claimToRoleMapper.mappings,
	}, nil
}

func validateConfigConsistency(configs []*storage.AuthMachineToMachineConfig) error {
	if len(configs) == 0 {
		return errox.InvalidArgs.New("no auth machine to machine configuration provided for exchanger creation")
	}
	issuer := configs[0].GetIssuer()
	exchangerType := configs[0].GetType()
	var multiErr error
	for _, config := range configs[1:] {
		cfgIssuer := config.GetIssuer()
		cfgType := config.GetType()
		if issuer != cfgIssuer {
			multiErr = multierror.Append(
				multiErr,
				errox.InvalidArgs.Newf("issuer mismatch %q vs %q", issuer, cfgIssuer),
			)
		}
		if exchangerType != cfgType {
			multiErr = multierror.Append(
				multiErr,
				errox.InvalidArgs.Newf("type mismatch %q vs %q", exchangerType.String(), cfgType.String()),
			)
		}
	}

	if multiErr != nil {
		return multiErr
	}
	return nil
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
	idTokens := make([]*IDToken, 0, len(m.verifiers))
	var subject string
	var multiErr error
	var verifiedAudiences set.StringSet
	for _, verifier := range m.verifiers {
		verifiedIDToken, err := verifier.VerifyIDToken(ctx, rawIDToken)
		if err != nil {
			multiErr = multierror.Append(multiErr, err)
			continue
		}
		subject = verifiedIDToken.Subject
		verifiedAudiences.Add(verifier.Audience())
		idTokens = append(idTokens, verifiedIDToken)
	}
	if len(idTokens) == 0 {
		return "", errox.NoCredentials.New("ID token is invalid").CausedBy(multiErr.Error())
	}

	log.Debugf("Successfully validated ID token (sub=%q) for issuer %q", subject, m.issuer)

	extractedClaims := make(map[string][]string)
	for _, verifiedIDToken := range idTokens {
		tokenClaims, err := m.roxClaimExtractor.ExtractClaims(verifiedIDToken)
		if err != nil {
			return "", errox.NoCredentials.New("extracting claims from ID token").CausedBy(err)
		}
		mergeClaims(extractedClaims, tokenClaims)
	}
	pruneDuplicateClaims(extractedClaims)
	// Only keep audience values that verified
	extractedClaims["aud"] = slices.DeleteFunc(extractedClaims["aud"], func(audience string) bool {
		return !verifiedAudiences.Contains(audience)
	})

	log.Debugf("String claims of the ID token (sub=%q): %+v", subject, extractedClaims)

	// We attempt to resolve the roles for the claims here. In case no roles are given for the ID token, we reject it.
	// Additionally, since the context will have no access, elevate it locally to fetch roles.
	resolveRolesCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))
	_, err := resolveRolesForClaims(resolveRolesCtx, extractedClaims, m.roleDS, m.mappings)
	if err != nil {
		return "", errox.NoCredentials.New("resolving roles for id token").CausedBy(err)
	}

	roxClaims, err := m.roxClaimExtractor.ExtractRoxClaims(extractedClaims)
	if err != nil {
		return "", errox.NoCredentials.New("creating claims for Central token").CausedBy(err)
	}

	log.Debugf("Rox claims for the ID token (sub=%q): %+v", subject, roxClaims)

	tokenInfo, err := m.issuer.Issue(ctx, roxClaims)
	if err != nil {
		return "", errox.NoCredentials.New("issuing Central token").CausedBy(err)
	}

	return tokenInfo.Token, nil
}

func mergeClaims(into map[string][]string, from map[string][]string) {
	for key, values := range from {
		if _, found := into[key]; !found {
			into[key] = make([]string, 0)
		}
		into[key] = append(into[key], values...)
	}
}

func pruneDuplicateClaims(claims map[string][]string) {
	for key, values := range claims {
		valueSet := set.NewStringSet(values...)
		claims[key] = valueSet.AsSortedSlice(func(i string, j string) bool { return i < j })
	}
}

func (m *machineToMachineTokenExchanger) Config() *storage.AuthMachineToMachineConfig {
	if len(m.configs) == 0 {
		return nil
	}
	return m.configs[0]
}

func (m *machineToMachineTokenExchanger) GetTokenExpirationDuration() time.Duration {
	return m.tokenTTL
}

func (m *machineToMachineTokenExchanger) Configs() []*storage.AuthMachineToMachineConfig {
	return m.configs
}

func (m *machineToMachineTokenExchanger) GetType() storage.AuthMachineToMachineConfig_Type {
	return m.exchangerType
}
