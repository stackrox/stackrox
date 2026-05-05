package m2m

import (
	"context"
	"regexp"
	"time"

	"github.com/pkg/errors"
	roleDataStore "github.com/stackrox/rox/central/role/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/authproviders"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/sac/resources"
	"github.com/stackrox/rox/pkg/uuid"
)

var (
	_ TokenExchanger = (*machineToMachineTokenExchanger)(nil)
)

const (
	m2mIssuerNamespace = "machine-to-machine-issuer"
)

// mapping clones storage.AuthMachineToMachineConfig_Mapping with a compiled regexp.
type mapping struct {
	key             string
	valueExpression string
	role            string
	expression      *regexp.Regexp
}

// TokenExchanger will exchange a raw ID token to a Rox token (i.e. a Central access token).
// This will be done based on one or more auth machine to machine configs.
//
//go:generate mockgen-wrapper
type TokenExchanger interface {
	ExchangeToken(ctx context.Context, rawIDToken string) (string, error)
	Provider() authproviders.Provider
	Configs() []*storage.AuthMachineToMachineConfig
	TokenTTL() time.Duration
}

type machineToMachineTokenExchanger struct {
	configs           []*storage.AuthMachineToMachineConfig
	mappings          []*mapping
	tokenTTL          time.Duration
	verifier          tokenVerifier
	provider          authproviders.Provider
	issuer            tokens.Issuer
	roleDS            roleDataStore.DataStore
	roxClaimExtractor claimExtractor
}

// newTokenExchanger creates a new token exchanger based on auth machine to machine configs.
func newTokenExchanger(
	ctx context.Context,
	configs []*storage.AuthMachineToMachineConfig,
	roleDS roleDataStore.DataStore,
	issuerFactory tokens.IssuerFactory,
) (TokenExchanger, error) {
	// Use the first config for initialization (backward compatibility)
	config := configs[0]
	configType := config.GetType()
	configTypeString := configType.String()
	configIssuer := config.GetIssuer()
	configID := uuid.NewV5FromNonUUIDs(m2mIssuerNamespace, configIssuer).String()

	tokenTTL, err := time.ParseDuration(config.GetTokenExpirationDuration())
	// Technically, this shouldn't happen, as the config is expected to be validated beforehand (i.e. when added to the
	// data store).
	if err != nil {
		return nil, errors.Wrap(err, "parsing token expiration duration")
	}

	mappings := compileMappings(config.GetMappings())

	verifier, err := tokenVerifierFromConfig(ctx, configType, configIssuer)
	if err != nil {
		return nil, err
	}
	provider := newProviderFromConfig(configID, configTypeString, newRoleMapper(roleDS, mappings))
	roxClaimExtractor := newClaimExtractorForType(configType)
	issuer, err := issuerFactory.CreateIssuer(provider, tokens.WithTTL(tokenTTL))
	if err != nil {
		return nil, errors.Wrap(err, "creating token issuer")
	}

	return &machineToMachineTokenExchanger{
		configs:           configs,
		mappings:          mappings,
		tokenTTL:          tokenTTL,
		issuer:            issuer,
		verifier:          verifier,
		provider:          provider,
		roxClaimExtractor: roxClaimExtractor,
		roleDS:            roleDS,
	}, nil
}

func (m *machineToMachineTokenExchanger) Provider() authproviders.Provider {
	return m.provider
}

func (m *machineToMachineTokenExchanger) ExchangeToken(ctx context.Context, rawIDToken string) (string, error) {
	idToken, err := m.verifier.VerifyIDToken(ctx, rawIDToken)
	if err != nil {
		return "", errox.NoCredentials.New("ID token is invalid").CausedBy(err)
	}

	log.Debugf("Successfully validated ID token (sub=%q) for config %q", idToken.Subject, m.configs[0].GetId())

	claims, err := m.roxClaimExtractor.ExtractClaims(idToken)
	if err != nil {
		return "", errox.NoCredentials.New("extracting claims from ID token").CausedBy(err)
	}

	log.Debugf("String claims of the ID token (sub=%q): %+v", idToken.Subject, claims)

	// We attempt to resolve the roles for the claims here. In case no roles are given for the ID token, we reject it.
	// Additionally, since the context will have no access, elevate it locally to fetch roles.
	resolveRolesCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))
	_, err = resolveRolesForClaims(resolveRolesCtx, claims, m.roleDS, m.mappings)
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

func (m *machineToMachineTokenExchanger) Configs() []*storage.AuthMachineToMachineConfig {
	return m.configs
}

func (m *machineToMachineTokenExchanger) TokenTTL() time.Duration {
	return m.tokenTTL
}

// compileMappings converts storage mappings to internal mappings with compiled regexps.
func compileMappings(storageMappings []*storage.AuthMachineToMachineConfig_Mapping) []*mapping {
	mappings := make([]*mapping, 0, len(storageMappings))

	for _, m := range storageMappings {
		// The mapping value is validated on insert / update to contain a valid regexp, thus we can use MustCompile here.
		mappings = append(mappings, &mapping{
			key:             m.GetKey(),
			valueExpression: m.GetValueExpression(),
			role:            m.GetRole(),
			expression:      regexp.MustCompile(m.GetValueExpression()),
		})
	}

	return mappings
}
