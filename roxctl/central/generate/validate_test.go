package generate

import (
	"testing"

	"github.com/stackrox/rox/pkg/renderer"
	"github.com/stretchr/testify/assert"
)

func newHostPathInstance(path, nodeSelectorKey, nodeSelectorValue string) *renderer.HostPathPersistenceInstance {
	return &renderer.HostPathPersistenceInstance{
		HostPath:          path,
		NodeSelectorKey:   nodeSelectorKey,
		NodeSelectorValue: nodeSelectorValue,
	}
}

func TestValidateHostPathInstance(t *testing.T) {
	cases := []struct {
		name        string
		instance    *renderer.HostPathPersistenceInstance
		expectedErr bool
	}{
		{
			name:        "nil",
			instance:    nil,
			expectedErr: false,
		},
		{
			name:        "empty",
			instance:    newHostPathInstance("", "", ""),
			expectedErr: true,
		},
		{
			name:        "path-only",
			instance:    newHostPathInstance("/var/lib/stackrox", "", ""),
			expectedErr: false,
		},
		{
			name:        "path-only-selector-key",
			instance:    newHostPathInstance("/var/lib/stackrox", "key", ""),
			expectedErr: true,
		},
		{
			name:        "path-only-selector-value",
			instance:    newHostPathInstance("/var/lib/stackrox", "", "value"),
			expectedErr: true,
		},
		{
			name:        "path-full-selector",
			instance:    newHostPathInstance("/var/lib/stackrox", "key", "value"),
			expectedErr: false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.expectedErr {
				assert.Error(t, validateHostPathInstance(c.instance))
			} else {
				assert.NoError(t, validateHostPathInstance(c.instance))
			}
		})
	}
}

func TestValidateHostPath(t *testing.T) {
	cases := []struct {
		name        string
		hostPath    *renderer.HostPathPersistence
		expectedErr bool
	}{
		{
			name:        "nil",
			hostPath:    nil,
			expectedErr: false,
		},
		{
			name:        "empty",
			hostPath:    &renderer.HostPathPersistence{},
			expectedErr: false,
		},
		{
			name: "central-only",
			hostPath: &renderer.HostPathPersistence{
				Central: newHostPathInstance("/var/lib/stackrox", "", ""),
			},
			expectedErr: false,
		},
		{
			name: "db-only",
			hostPath: &renderer.HostPathPersistence{
				DB: newHostPathInstance("/var/lib/centraldb", "", ""),
			},
			expectedErr: false,
		},
		{
			name: "both",
			hostPath: &renderer.HostPathPersistence{
				Central: newHostPathInstance("/var/lib/stackrox", "", ""),
				DB:      newHostPathInstance("/var/lib/centraldb", "", ""),
			},
			expectedErr: false,
		},
		{
			name: "error-on-central",
			hostPath: &renderer.HostPathPersistence{
				Central: newHostPathInstance("/var/lib/stackrox", "key", ""),
				DB:      newHostPathInstance("/var/lib/centraldb", "", ""),
			},
			expectedErr: true,
		},
		{
			name: "error-on-db",
			hostPath: &renderer.HostPathPersistence{
				Central: newHostPathInstance("/var/lib/stackrox", "", ""),
				DB:      newHostPathInstance("/var/lib/centraldb", "key", ""),
			},
			expectedErr: true,
		},
		{
			name: "error-on-both",
			hostPath: &renderer.HostPathPersistence{
				Central: newHostPathInstance("/var/lib/stackrox", "key", ""),
				DB:      newHostPathInstance("/var/lib/centraldb", "key", ""),
			},
			expectedErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.expectedErr {
				assert.Error(t, validateHostPath(c.hostPath))
			} else {
				assert.NoError(t, validateHostPath(c.hostPath))
			}
		})
	}
}
