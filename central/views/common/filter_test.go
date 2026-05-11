package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplyActiveDeploymentExclusion_EmptyWhere(t *testing.T) {
	where, values := ApplyActiveDeploymentExclusion("", nil, "images.id", "image_id")

	assert.Contains(t, where, "NOT EXISTS")
	assert.Contains(t, where, "dc.image_id = images.id")
	assert.Contains(t, where, "d.state = $$")
	assert.Equal(t, []interface{}{int32(0)}, values)
}

func TestApplyActiveDeploymentExclusion_ExistingWhere(t *testing.T) {
	where, values := ApplyActiveDeploymentExclusion(
		"images.name = $$", []interface{}{"nginx"}, "images.id", "image_id",
	)

	assert.Contains(t, where, "(images.name = $$) and NOT EXISTS")
	assert.Equal(t, []interface{}{"nginx", int32(0)}, values)
}

func TestApplyActiveDeploymentExclusion_V2Columns(t *testing.T) {
	where, values := ApplyActiveDeploymentExclusion("", nil, "images_v2.id", "image_idv2")

	assert.Contains(t, where, "dc.image_idv2 = images_v2.id")
	assert.Equal(t, []interface{}{int32(0)}, values)
}

func TestApplyActiveDeploymentExclusion_DoesNotMutateInput(t *testing.T) {
	original := []interface{}{"a", "b"}
	origCopy := make([]interface{}, len(original))
	copy(origCopy, original)

	_, _ = ApplyActiveDeploymentExclusion("x = $$", original, "images.id", "image_id")

	assert.Equal(t, origCopy, original, "input slice must not be mutated")
}
