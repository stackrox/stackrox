package token

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/continuousintegration/datastore"
	"github.com/stackrox/rox/central/jwt"
	"github.com/stackrox/rox/central/role/resources"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/sac"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/stringutils"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log      = logging.LoggerForModule()
	once     sync.Once
	instance Exchanger
)

// Exchanger allows to exchange an ID token from a continuous integration provider for an access token.
type Exchanger interface {
	ExchangeToken(ctx context.Context, idToken string, integrationType storage.ContinuousIntegrationType) (string, error)
}

// Singleton returns the Exchanger singleton.
func Singleton() Exchanger {
	once.Do(func() {
		issuer, err := jwt.IssuerFactorySingleton().CreateIssuer(
			SingletonSourceForContinuousIntegration(storage.ContinuousIntegrationType_GITHUB_ACTIONS),
			tokens.WithTTL(2*time.Hour))
		utils.Must(err)
		issuers := map[storage.ContinuousIntegrationType]tokens.Issuer{
			storage.ContinuousIntegrationType_GITHUB_ACTIONS: issuer,
		}
		instance = &retriever{continuousIntegrationDataStore: datastore.Singleton(), issuers: issuers}
	})
	return instance
}

type retriever struct {
	issuers                        map[storage.ContinuousIntegrationType]tokens.Issuer
	continuousIntegrationDataStore datastore.DataStore
}

// ExchangeToken exchanges a token based on an ID token from a continuous integration provider. This is done by:
// a) verify that the ID token is issued by the specific continuous integration provider.
// b) based on the configured continuous integration configs, obtain the rules the token should have.
// c) issue a token with a lower TTL (currently 2 hours) and return the access token.
//
// In case no roles are found for the specific continuous integration provider, no token will be issued and an error
// will be returned.
func (r *retriever) ExchangeToken(ctx context.Context, rawIDToken string, integrationType storage.ContinuousIntegrationType) (string, error) {
	idToken, err := r.verifyIDToken(ctx, rawIDToken, integrationType)
	if err != nil {
		return "", errors.Wrap(err, "verifying id token")
	}
	log.Infof("Verified the ID token from %s: %+v", integrationType, idToken)

	var claims map[string]interface{}
	utils.Should(idToken.Claims(&claims))
	log.Infof("Claims of the ID token from %s: %+v", integrationType, claims)

	configsForType, err := r.getConfigsForType(ctx, integrationType)
	if err != nil {
		return "", errors.Wrapf(err, "getting configs for type %s", integrationType)
	}
	log.Infof("Goet the following configs for type %s: %+v", integrationType, configsForType)

	rolesForIDToken := r.getRoles(configsForType, idToken)
	log.Infof("Got the following roles for the ID token (sub=%s) from %s: [%s]",
		idToken.Subject, integrationType.String(), strings.Join(rolesForIDToken, ","))

	if len(rolesForIDToken) == 0 {
		return "", errox.NotAuthorized.Newf("no roles configured to use for type %s", integrationType)
	}

	roxClaims, err := r.getRoxClaims(idToken, rolesForIDToken)
	if err != nil {
		return "", errors.Wrap(err, "creating rox claims from ID token")
	}

	issuer, exists := r.issuers[integrationType]
	if !exists {
		return "", errox.NotFound.Newf("no token issuer available for type %s", integrationType)
	}

	tokenInfo, err := issuer.Issue(ctx, roxClaims)
	if err != nil {
		return "", errors.Wrap(err, "issuing token")
	}
	return tokenInfo.Token, nil
}

func (r *retriever) verifyIDToken(ctx context.Context, rawIDToken string, integrationType storage.ContinuousIntegrationType) (*oidc.IDToken, error) {
	var (
		provider *oidc.Provider
	)
	switch integrationType {
	case storage.ContinuousIntegrationType_GITHUB_ACTIONS:
		gh, err := oidc.NewProvider(ctx, "https://token.actions.githubusercontent.com")
		if err != nil {
			return nil, errors.Wrap(err, "creating OIDC provider for GitHub")
		}
		provider = gh
	default:
		return nil, errox.InvalidArgs.Newf("cannot verify ID token for CI type %s", integrationType.String())
	}
	verifier := provider.Verifier(&oidc.Config{
		// The audience is non-deterministic as it will be the repository owner's URL (which may be a magnitude of URLs)
		// and can be customized, hence skipping this.
		// See https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#understanding-the-oidc-token
		SkipClientIDCheck: true,
	})
	return verifier.Verify(ctx, rawIDToken)
}

func (r *retriever) getRoles(configs []*storage.ContinuousIntegrationConfig, idToken *oidc.IDToken) []string {
	rolesToAssign := set.NewStringSet()
	for _, cfg := range configs {
		for _, mapping := range cfg.GetMappings() {
			if valuesMatch(idToken.Subject, mapping.GetValue()) && !rolesToAssign.Contains(mapping.GetRole()) {
				rolesToAssign.Add(mapping.GetRole())
			}
		}
	}
	return rolesToAssign.AsSlice()
}

func (r *retriever) getConfigsForType(ctx context.Context, integrationType storage.ContinuousIntegrationType) ([]*storage.ContinuousIntegrationConfig, error) {
	configCtx := sac.WithGlobalAccessScopeChecker(ctx,
		sac.AllowFixedScopes(
			sac.AccessModeScopeKeys(storage.Access_READ_ACCESS),
			sac.ResourceScopeKeys(resources.Role, resources.Access)))
	configs, err := r.continuousIntegrationDataStore.GetAllContinuousIntegrationConfigs(configCtx)
	if err != nil {
		return nil, err
	}

	configsForType := make([]*storage.ContinuousIntegrationConfig, 0, len(configs))

	for _, config := range configs {
		if config.GetType() == integrationType {
			configsForType = append(configsForType, config)
		}
	}
	return configsForType, nil
}

type githubClaims struct {
	RunID       string `json:"run_id,omitempty"`
	Repository  string `json:"repository,omitempty"`
	WorkflowRef string `json:"workflow_ref,omitempty"`
	Actor       string `json:"actor,omitempty"`
	ActorID     string `json:"actor_id,omitempty"`
}

func (r *retriever) getRoxClaims(idToken *oidc.IDToken, roles []string) (tokens.RoxClaims, error) {
	var claims githubClaims
	if err := idToken.Claims(&claims); err != nil {
		return tokens.RoxClaims{}, err
	}

	actorID := utils.IfThenElse(claims.ActorID != "", "|"+claims.ActorID, "")
	userClaims := &tokens.ExternalUserClaim{
		UserID:   fmt.Sprintf("%s|%s%s", claims.WorkflowRef, idToken.Audience, actorID),
		FullName: stringutils.FirstNonEmpty(claims.Actor, "GitHub Actions"),
		Attributes: map[string][]string{
			"run_id":     {claims.RunID},
			"actor":      {claims.Actor},
			"actor_id":   {claims.ActorID},
			"repository": {claims.Repository},
		},
	}

	return tokens.RoxClaims{
		RoleNames:    roles,
		ExternalUser: userClaims,
		Name:         fmt.Sprintf("GitHubActions%s", actorID),
	}, nil
}

func checkIfRegexp(expr string) *regexp.Regexp {
	parsedExpr, err := regexp.Compile(expr)
	if err != nil {
		return nil
	}
	return parsedExpr
}

func valuesMatch(claimValue string, expr string) bool {
	// The expression is either a simple string value or a regular expression.
	if regExpr := checkIfRegexp(expr); regExpr != nil {
		return regExpr.MatchString(claimValue)
	}
	// Otherwise if it is not a regular expression, fall back to string comparison.
	return claimValue == expr
}
