package printer

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestEnvUpToDate(t *testing.T) {
	for k, v := range storage.ContainerConfig_EnvironmentConfig_EnvVarSource_value {
		asSrc := storage.ContainerConfig_EnvironmentConfig_EnvVarSource(v)
		if asSrc == storage.ContainerConfig_EnvironmentConfig_UNSET || asSrc == storage.ContainerConfig_EnvironmentConfig_UNKNOWN {
			continue
		}
		assert.Contains(t, envVarSourceToNameMap, k)
	}
}
