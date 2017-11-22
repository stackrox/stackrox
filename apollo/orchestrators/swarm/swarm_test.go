// +build swarm

package swarm

import (
	"testing"

	"bitbucket.org/stack-rox/apollo/apollo/orchestrators/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSwarm(t *testing.T) {
	orchestrator, err := newSwarm()
	require.Nil(t, err)
	require.NotNil(t, orchestrator)
}

func TestLaunch(t *testing.T) {
	orchestrator, err := newSwarm()
	require.Nil(t, err)

	service := types.SystemService{
		Envs:   []string{"ROX_APOLLO_ENDPOINT=localhost:8080"},
		Image:  "stackrox/docker-bench-bootstrap:latest",
		Mounts: []string{"/var/run/docker.sock:/var/run/docker.sock"},
		Global: true,
	}

	_, err = orchestrator.Launch(service)
	assert.Nil(t, err)
}
