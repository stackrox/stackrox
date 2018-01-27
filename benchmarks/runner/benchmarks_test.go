package runner

import (
	"os"
	"testing"

	"bitbucket.org/stack-rox/apollo/pkg/checks"
	"bitbucket.org/stack-rox/apollo/pkg/checks/host_configuration"
	"bitbucket.org/stack-rox/apollo/pkg/checks/utils"
	"bitbucket.org/stack-rox/apollo/pkg/env"
	"github.com/stretchr/testify/assert"
)

func TestRenderChecks(t *testing.T) {
	expectedChecks := []utils.Check{
		hostconfiguration.NewContainerPartitionBenchmark(),
		hostconfiguration.NewHostHardened(),
	}
	assert.NoError(t, os.Setenv(env.Checks.EnvVar(), "CIS 1.1,CIS 1.2"))
	checks := renderChecks()
	assert.Equal(t, expectedChecks, checks)
}

func TestRegistry(t *testing.T) {
	reg := checks.Registry
	assert.NotEmpty(t, reg)
}
