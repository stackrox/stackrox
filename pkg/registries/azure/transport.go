package azure

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/containers/azcontainerregistry"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/urlfmt"
)

const (
	earlyExpiry = 5 * time.Minute

	// 000... is the generic username that must be used when converting ACR refresh tokens to docker login credentials.
	oauthUsername = "00000000-0000-0000-0000-000000000000"
)

type azureTransport struct {
	registry.Transport
	name        string
	config      *docker.Config
	serviceName string
	creds       *azidentity.DefaultAzureCredential
	client      *azcontainerregistry.AuthenticationClient
	expiresAt   *time.Time
	mutex       sync.RWMutex
}

func newAzureTransport(name string, config *docker.Config,
	creds *azidentity.DefaultAzureCredential, client *azcontainerregistry.AuthenticationClient,
) *azureTransport {
	// The service name must be of the form `registry.azurecr.io` without scheme or slash.
	serviceName := urlfmt.FormatURL(config.Endpoint, urlfmt.NONE, urlfmt.NoTrailingSlash)
	transport := &azureTransport{name: name, config: config, serviceName: serviceName, creds: creds, client: client}
	ctx := context.Background()
	if err := transport.refreshNoLock(ctx); err != nil {
		log.Error("Failed to refresh Azure container registry token: ", err)
	}
	return transport
}

func (t *azureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// We perform a TOC-TOU intentionally to optimize the read path.
	// This is advantageous because...
	// a) we only need a write lock every 3 hours to refresh the token.
	// b) refreshing the token multiple times is idempotent.
	// c) we do not want to block the entire read path for performance reasons.
	if err := t.ensureValid(req.Context()); err != nil {
		return nil, err
	}
	return concurrency.WithRLock2(&t.mutex,
		func() (*http.Response, error) { return t.Transport.RoundTrip(req) },
	)
}

// ensureValid refreshes the access token if it is invalid.
func (t *azureTransport) ensureValid(ctx context.Context) error {
	if concurrency.WithRLock1(&t.mutex, t.isValidNoLock) {
		return nil
	}
	return concurrency.WithLock1(&t.mutex, func() error { return t.refreshNoLock(ctx) })
}

func (t *azureTransport) isValidNoLock() bool {
	return t.expiresAt != nil && time.Now().Before(t.expiresAt.Add(-earlyExpiry))
}

func (t *azureTransport) refreshNoLock(ctx context.Context) error {
	log.Debugf("Refreshing Azure container registry token for image integration %q", t.name)
	// First obtain an Azure AD access token via the default credentials chain.
	aadToken, err := t.creds.GetToken(ctx,
		policy.TokenRequestOptions{Scopes: []string{"https://management.azure.com/.default"}},
	)
	if err != nil {
		return errors.Wrap(err, "getting Azure access token")
	}
	// Then exchange the AAD access token for an ACR refresh token. The refresh token is also a valid
	// docker login password.
	rtResp, err := t.client.ExchangeAADAccessTokenForACRRefreshToken(ctx,
		azcontainerregistry.PostContentSchemaGrantTypeAccessToken,
		t.serviceName,
		&azcontainerregistry.AuthenticationClientExchangeAADAccessTokenForACRRefreshTokenOptions{
			AccessToken: &aadToken.Token,
		},
	)
	if err != nil {
		return errors.Wrap(err, "getting Azure container registry refresh token")
	}
	// ACR refresh token are valid for three hours from the time of exchange.
	rtExpiry := time.Now().Add(3 * time.Hour)
	t.expiresAt = &rtExpiry
	t.config.SetCredentials(oauthUsername, *rtResp.RefreshToken)
	t.Transport = docker.DefaultTransport(t.config)
	return nil
}
