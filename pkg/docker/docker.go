package docker

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"bitbucket.org/stack-rox/apollo/pkg/logging"
	"github.com/docker/docker/client"
)

var (
	log = logging.New("pkg/docker")
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
	client, err := client.NewEnvClient()
	if err != nil {
		return nil, fmt.Errorf("Unable to create docker client: %+v", err)
	}
	return client, nil
}

func getDockerVersion(v string) (float64, error) {
	version, err := strconv.ParseFloat(v, 64)
	return version, err
}

func dockerVersionString(v float64) string {
	return fmt.Sprintf("%0.2f", v)
}

// TimeoutContext returns a context with a timeout with the duration of the hang timeout
func TimeoutContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), HangTimeout)
}

// NegotiateClientVersionToLatest negotiates the golang API version with the Docker server
func NegotiateClientVersionToLatest(client client.APIClient, dockerAPIVersion float64) error {
	// update client version to lowest supported version
	client.UpdateClientVersion(dockerVersionString(DefaultAPIVersion))
	versionStruct, err := client.ServerVersion(context.Background())
	if err != nil {
		return fmt.Errorf("unable to get docker server version: %+v", err)
	}
	var minClientVersion float64
	if versionStruct.MinAPIVersion == "" { // Backwards compatibility
		minClientVersion, err = getDockerVersion(versionStruct.APIVersion)
		if err != nil {
			return fmt.Errorf("unable to parse docker server api version: %+v", err)
		}
	} else {
		minClientVersion, err = getDockerVersion(versionStruct.MinAPIVersion)
		if err != nil {
			return fmt.Errorf("unable to parse docker server min api version: %+v", err)
		}
	}
	versionToNegotiate := dockerAPIVersion
	if dockerAPIVersion < minClientVersion {
		versionToNegotiate = minClientVersion
	}
	log.Infof("Negotiating Docker API version to %v", versionToNegotiate)
	client.UpdateClientVersion(dockerVersionString(versionToNegotiate))
	return nil
}
