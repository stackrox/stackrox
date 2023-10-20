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
}

type machineToMachineTokenExchanger struct {
	config            *storage.AuthMachineToMachineConfig
	configRegExps     []*regexp.Regexp
	verifier          tokenVerifier
	provider          authproviders.Provider
	issuer            tokens.Issuer
	roleDS            roleDataStore.DataStore
	roxClaimExtractor claimExtractor
}

// newTokenExchanger creates a new token exchanger based on an auth machine to machine config.
func newTokenExchanger(ctx context.Context, config *storage.AuthMachineToMachineConfig,
	roleDS roleDataStore.DataStore, issuerFactory tokens.IssuerFactory) (TokenExchanger, error) {
	tokenTTL, err := time.ParseDuration(config.GetTokenExpirationDuration())
	// Technically, this shouldn't happen, as the config is expected to be validated beforehand (i.e. when added to the
	// data store).
	if err != nil {
		return nil, errors.Wrap(err, "parsing token expiration duration")
	}

	configRegExps := createRegexp(config)

	verifier, err := tokenVerifierFromConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	provider := newProviderFromConfig(config, newRoleMapper(config, roleDS, configRegExps))
	roxClaimExtractor := newClaimExtractorFromConfig(config)
	issuer, err := issuerFactory.CreateIssuer(provider, tokens.WithTTL(tokenTTL))
	if err != nil {
		return nil, errors.Wrap(err, "creating token issuer")
	}

	return &machineToMachineTokenExchanger{
		config:            config,
		configRegExps:     configRegExps,
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

	log.Debugf("Successfully validated ID token (sub=%q) for config %q", idToken.Subject, m.config.GetId())

	var unstructured map[string]interface{}
	if err := idToken.Claims(&unstructured); err != nil {
		return "", errox.NoCredentials.New("extracting claims from ID token").CausedBy(err)
	}

	log.Debugf("Unstructured claims of the ID token(sub=%q): %+v", idToken.Subject, unstructured)

	// We currently only support non-nested claims to be used within mappings.
	claims := mapToStringClaims(unstructured)

	log.Debugf("String claims of the ID token (sub=%q): %+v", idToken.Subject, claims)

	// We attempt to resolve the roles for the claims here. In case no roles are given for the ID token, we reject it.
	// Additionally, since the context will have no access, elevate it locally to fetch roles.
	resolveRolesCtx := sac.WithGlobalAccessScopeChecker(ctx, sac.AllowFixedScopes(
		sac.AccessModeScopeKeys(storage.Access_READ_ACCESS), sac.ResourceScopeKeys(resources.Access)))
	_, err = resolveRolesForClaims(resolveRolesCtx, claims, m.roleDS, m.config.GetMappings(), m.configRegExps)
	if err != nil {
		return "", errox.NoCredentials.New("resolving roles for id token").CausedBy(err)
	}

	roxClaims, err := m.roxClaimExtractor.ExtractRoxClaims(idToken)
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

func mapToStringClaims(claims map[string]interface{}) map[string][]string {
	stringClaims := make(map[string][]string, len(claims))
	for key, value := range claims {
		switch value := value.(type) {
		case string:
			stringClaims[key] = []string{value}
		case []string:
			stringClaims[key] = value
		default:
			log.Debugf("Dropping value %v for claim %s since its a nested claim or a non-string type %T", value, key, value)
		}
	}

	return stringClaims
}

func createRegexp(config *storage.AuthMachineToMachineConfig) []*regexp.Regexp {
	regExps := make([]*regexp.Regexp, 0, len(config.GetMappings()))

	for _, mapping := range config.GetMappings() {
		// The mapping value is validated on insert / update to contain a valid regexp, thus we can use MustCompile here.
		regExps = append(regExps, regexp.MustCompile(mapping.GetValue()))
	}

	return regExps
}
