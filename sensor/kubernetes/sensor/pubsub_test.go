package sensor

import (
	"fmt"
	"testing"

	"github.com/stackrox/rox/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPubSubDispatcher(t *testing.T) {
	tests := map[string]struct {
		releaseBuild           bool
		concurrentLanesEnabled bool
	}{
		"release build uses concurrent lanes": {
			releaseBuild: true,
		},
		"release build ignores env var set to false": {
			releaseBuild:           true,
			concurrentLanesEnabled: false,
		},
		"dev build defaults to blocking lanes": {
			releaseBuild: false,
		},
		"dev build with env var uses concurrent lanes": {
			releaseBuild:           false,
			concurrentLanesEnabled: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Setenv(env.PubSubConcurrentLanes.EnvVar(), fmt.Sprintf("%t", tt.concurrentLanesEnabled))
			dispatcher, err := buildPubSubDispatcher(tt.releaseBuild)
			require.NoError(t, err)
			assert.NotNil(t, dispatcher)
			dispatcher.Stop()
		})
	}
}
