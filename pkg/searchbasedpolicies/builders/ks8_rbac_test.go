package builders

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestK8sRBACMap(t *testing.T) {
	for levelInt := range storage.PermissionLevel_name {
		level := storage.PermissionLevel(levelInt)
		if level == storage.PermissionLevel_UNSET || level == storage.PermissionLevel_NONE {
			continue
		}
		_, ok := rbacPermissionLabels[level]
		assert.True(t, true, ok)
	}
}
