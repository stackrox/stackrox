package azure

import (
	"context"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/sync"
)

const earlyExpiry = 5 * time.Minute

type azureTransport struct {
	registry.Transport
	name      string
	config    *docker.Config
	creds     *azidentity.DefaultAzureCredential
	expiresAt *time.Time
	mutex     sync.RWMutex
}

func newAzureTransport(name string, config *docker.Config, creds *azidentity.DefaultAzureCredential) *azureTransport {
	transport := &azureTransport{name: name, config: config, creds: creds}
	ctx := context.Background()
	if err := transport.refreshNoLock(ctx); err != nil {
		log.Error("Failed to refresh ACR token: ", err)
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
	if !concurrency.WithRLock1(&t.mutex, t.isValidNoLock) {
		if err := concurrency.WithLock1(&t.mutex, func() error { return t.refreshNoLock(ctx) }); err != nil {
			return err
		}
	}
	return nil
}

func (t *azureTransport) isValidNoLock() bool {
	return t.expiresAt != nil && time.Now().Before(t.expiresAt.Add(-earlyExpiry))
}

func (t *azureTransport) refreshNoLock(ctx context.Context) error {
	log.Debugf("Refreshing ACR token for image integration %q", t.name)
	token, err := t.creds.GetToken(ctx, policy.TokenRequestOptions{})
	if err != nil {
		return errors.Wrap(err, "getting Azure access token")
	}
	t.expiresAt = &token.ExpiresOn
	t.config.SetCredentials("00000000-0000-0000-0000-000000000000", token.Token)
	t.Transport = docker.DefaultTransport(t.config)
	return nil
}
