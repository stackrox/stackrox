// +build swarm

package swarm

import (
	"testing"

	"github.com/stackrox/rox/central/orchestrators/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSwarm(t *testing.T) {
	orchestrator, err := New()
	require.Nil(t, err)
	require.NotNil(t, orchestrator)
}

func TestLaunch(t *testing.T) {
	orchestrator, err := New()
	require.Nil(t, err)

	service := types.SystemService{
		Envs:    []string{"ROX_CENTRAL_ENDPOINT=localhost:443"},
		Image:   "stackrox/prevent:latest",
		Mounts:  []string{"/var/run/docker.sock:/var/run/docker.sock"},
		Global:  true,
		Command: []string{"benchmark-bootstrap"},
	}

	_, err = orchestrator.Launch(service)
	assert.Nil(t, err)
}
