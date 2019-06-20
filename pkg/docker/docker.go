package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/pkg/docker/client"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	// HangTimeout is the maximum length of time that we will wait for the docker client to respond
	HangTimeout = 30 * time.Second
)

// NewClient returns a new docker client or an error if there was issues generating it
func NewClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create docker client")
	}
	return cli, nil
}

// NewClientWithPath returns a docker client with the path
func NewClientWithPath(host string) (*client.Client, error) {
	return client.NewClient(host, api.DefaultVersion, nil, nil)
}

// TimeoutContext returns a context with a timeout with the duration of the hang timeout
func TimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), HangTimeout)
}
