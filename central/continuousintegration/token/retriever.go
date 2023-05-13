package token

import (
	"context"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/continuousintegration/datastore"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/auth/tokens"
	"github.com/stackrox/rox/pkg/errox"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/maputil"
	"github.com/stackrox/rox/pkg/utils"
)

var (
	log = logging.LoggerForModule()
)

type Retriever interface {
	RetrieveToken(ctx context.Context, idToken string, integrationType storage.ContinuousIntegrationType) (string, error)
}

type idTokenVerifier interface {
	VerifyToken(ctx context.Context, idToken string) (*oidc.IDToken, error)
}

type roleMapper interface {
	MapRoles(configsForType []*storage.ContinuousIntegrationConfig, idToken *oidc.IDToken) []string
}

func newRetriever() Retriever {

	return &retriever{
		v: map[string]idTokenVerifier{
			storage.ContinuousIntegrationType_GITHUB_ACTIONS.String(): newGitHubVerifier(),
		},
		m: &roleMapperImpl{},
	}
}

type retriever struct {
	v map[string]idTokenVerifier
	m roleMapper
	i tokens.Issuer
	d datastore.DataStore
}

func (r *retriever) RetrieveToken(ctx context.Context, rawIdToken string, integrationType storage.ContinuousIntegrationType) (string, error) {
	v, exists := r.v[integrationType.String()]
	if !exists {
		return "", errox.InvalidArgs.Newf(
			"cannot verify token for integration type %s since its not supported, choose one of [%s]",
			integrationType.String(), strings.Join(maputil.Keys(r.v), ","))
	}

	idToken, err := v.VerifyToken(ctx, rawIdToken)
	if err != nil {
		return "", errors.Wrap(err, "verifying id token")
	}
	log.Infof("Verified the ID token from %s: %+v", integrationType, idToken)

	// TODO(dhaus): Just for debugging / inspecting the token for the moment, remove later.
	var claims map[string]interface{}
	utils.Should(idToken.Claims(claims))
	log.Infof("Claims of the ID token from %s: %+v", integrationType, claims)

	configsForType, err := r.getConfigsForType(ctx, integrationType)
	if err != nil {
		return "", errors.Wrapf(err, "getting configs for type %s", integrationType)
	}
	log.Infof("Goet the following configs for type %s: %+v", integrationType, configsForType)

	rolesForIdToken := r.m.MapRoles(configsForType, idToken)
	log.Infof("Got the following roles for the ID token (sub=%s) from %s: [%s]",
		idToken.Subject, integrationType.String(), strings.Join(rolesForIdToken, ","))

	// TODO(dhaus): Map roles and issue token.
	return "", nil
}

func (r *retriever) getConfigsForType(ctx context.Context, integrationType storage.ContinuousIntegrationType) ([]*storage.ContinuousIntegrationConfig, error) {
	// TODO(dhaus): Need to potentially add privileges to context here.
	configs, err := r.d.GetAllContinuousIntegrationConfigs(ctx)
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
