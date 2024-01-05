package google

import (
	"net/http"

	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
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
	mutex       sync.Mutex
}

func newGoogleTransport(config *docker.Config, tokenSource oauth2.TokenSource) *googleTransport {
	transport := &googleTransport{config: config, tokenSource: tokenSource}
	if err := transport.refreshNoLock(); err != nil {
		log.Error("Failed to refresh token: ", err)
	}
	return transport
}

func (t *googleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if !t.token.Valid() {
		if err := t.refreshNoLock(); err != nil {
			return nil, err
		}
	}
	return t.Transport.RoundTrip(req)
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
