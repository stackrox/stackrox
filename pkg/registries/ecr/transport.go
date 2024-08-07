package ecr

import (
	"context"
	"encoding/base64"
	"net/http"
	"strings"
	"time"

	awsECR "github.com/aws/aws-sdk-go-v2/service/ecr"
	"github.com/heroku/docker-registry-client/registry"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/registries/docker"
	"github.com/stackrox/rox/pkg/sync"
)

const earlyExpiry = 5 * time.Minute

type awsTransport struct {
	registry.Transport
	name      string
	config    *docker.Config
	client    *awsECR.Client
	expiresAt *time.Time
	mutex     sync.RWMutex
}

func newAWSTransport(name string, config *docker.Config, client *awsECR.Client) *awsTransport {
	transport := &awsTransport{name: name, config: config, client: client}
	ctx := context.Background()
	if err := transport.refreshNoLock(ctx); err != nil {
		log.Error("Failed to refresh ECR token: ", err)
	}
	return transport
}

func (t *awsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// We perform a TOC-TOU intentionally to optimize the read path.
	// This is advantageous because...
	// a) we only need a write lock every 12 hours to refresh the token.
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
func (t *awsTransport) ensureValid(ctx context.Context) error {
	if !concurrency.WithRLock1(&t.mutex, t.isValidNoLock) {
		if err := concurrency.WithLock1(&t.mutex, func() error { return t.refreshNoLock(ctx) }); err != nil {
			return err
		}
	}
	return nil
}

func (t *awsTransport) isValidNoLock() bool {
	return t.expiresAt != nil && time.Now().Before(t.expiresAt.Add(-earlyExpiry))
}

func (t *awsTransport) refreshNoLock(ctx context.Context) error {
	log.Debugf("Refreshing ECR token for image integration %q", t.name)
	authToken, err := t.client.GetAuthorizationToken(ctx, &awsECR.GetAuthorizationTokenInput{})
	if err != nil {
		return errors.Wrap(err, "failed to get authorization token")
	}
	if len(authToken.AuthorizationData) == 0 {
		return errors.New("received empty authorization data in token")
	}
	authData := authToken.AuthorizationData[0]
	decoded, err := base64.StdEncoding.DecodeString(*authData.AuthorizationToken)
	if err != nil {
		return errors.Wrap(err, "failed to decode authorization token")
	}
	username, password, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return errors.New("malformed basic auth response from AWS")
	}
	t.expiresAt = authData.ExpiresAt
	t.config.SetCredentials(username, password)
	t.Transport = docker.DefaultTransport(t.config)
	return nil
}
