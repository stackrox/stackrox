package docker

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/docker/docker/client"
	"github.com/stackrox/rox/pkg/logging"
)

var (
	log = logging.LoggerForModule()
)

const (
	// DefaultAPIVersion is the Docker API version we will use in the
	// absence of an exact version we can detect at runtime.
	// This should be the API version for the minimum Docker version we support.
	// For Docker version to API version table, see:
	//   https://docs.docker.com/engine/reference/api/docker_remote_api/
	DefaultAPIVersion = 1.22
	// HangTimeout is the maximum length of time that we will wait for the docker client to respond
	HangTimeout = 30 * time.Second
)

// NewClient returns a new docker client or an error if there was issues generating it
func NewClient() (*client.Client, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Unable to create docker client: %+v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cli.NegotiateAPIVersion(ctx)
	return cli, nil
}

// TimeoutContext returns a context with a timeout with the duration of the hang timeout
func TimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), HangTimeout)
}

// IsContainerized returns true if the process calling it is within a container
func IsContainerized() bool {
	if os.Getenv("CIRCLECI") != "" {
		return false
	}
	data, err := ioutil.ReadFile("/proc/1/cgroup")
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte("docker"))
}
