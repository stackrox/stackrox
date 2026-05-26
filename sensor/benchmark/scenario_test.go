package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadSteadySyntheticDev(t *testing.T) {
	s, err := LoadScenario("benchmarks/sensor/scenarios/v0/steady-synthetic-dev")
	require.NoError(t, err)
	require.Equal(t, "steady-synthetic-dev-v0", s.Metadata.Name)
	require.FileExists(t, s.ResolvedWorkloadPath())
}
