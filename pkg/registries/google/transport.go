package google

import (
	"net/http"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/sync"
	"golang.org/x/oauth2"
)

var log = logging.LoggerForModule()

// googleTransport represents a transport that converts an oauth token source
// into docker registry credentials.
// This kind of trickery is required because the docker API does not
// accept a standard oauth2 transport.
type googleTransport struct {
	registry.Transport
	config      *docker.Config
	token       *oauth2.Token
	tokenSource oauth2.TokenSource
	mutex       sync.RWMutex
}

func newGoogleTransport(config *docker.Config, tokenSource oauth2.TokenSource) *googleTransport {
	transport := &googleTransport{config: config, tokenSource: tokenSource}
	if err := transport.refreshNoLock(); err != nil {
		log.Error("Failed to refresh token: ", err)
	}
	return transport
}

func (t *googleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// We perform a TOC-TOU intentionally to optimize the read path.
	// This is advantageous because...
	// a) we only need a write lock every hour to refresh the token.
	// b) refreshing the token multiple times is idempotent.
	// c) we do not want to block the entire read path for performance reasons.
	if !concurrency.WithRLock1(&t.mutex, t.token.Valid) {
		if err := concurrency.WithLock1(&t.mutex, t.refreshNoLock); err != nil {
			return nil, err
		}
	}
	return concurrency.WithRLock2(&t.mutex,
		func() (*http.Response, error) { return t.Transport.RoundTrip(req) },
	)
}

func (t *googleTransport) refreshNoLock() error {
	token, err := t.tokenSource.Token()
	if err != nil {
		return errors.Wrap(err, "failed to get access token")
	}
	t.token = token
	t.config.Username = "oauth2accesstoken"
	t.config.Password = token.AccessToken
	t.Transport = docker.DefaultTransport(t.config)
	return nil
}
