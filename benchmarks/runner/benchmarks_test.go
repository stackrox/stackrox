package runner

import (
	"os"
	"testing"

	"github.com/stackrox/rox/benchmarks/checks"
	"github.com/stackrox/rox/benchmarks/checks/host_configuration"
	"github.com/stackrox/rox/benchmarks/checks/utils"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
)

func TestRenderChecks(t *testing.T) {
	expectedChecks := []utils.Check{
		hostconfiguration.NewContainerPartitionBenchmark(),
		hostconfiguration.NewHostHardened(),
	}
	assert.NoError(t, os.Setenv(env.Checks.EnvVar(), "CIS Docker v1.1.0 - 1.1,CIS Docker v1.1.0 - 1.2"))
	checks := renderChecks()
	assert.Equal(t, expectedChecks, checks)
}

func TestRegistry(t *testing.T) {
	reg := checks.Registry
	assert.NotEmpty(t, reg)
}
