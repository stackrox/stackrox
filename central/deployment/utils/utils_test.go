package utils

import (
	"testing"

	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stretchr/testify/assert"
)

func TestGetMaskedDeploymentID(t *testing.T) {
	expected := "b289f32e-1a05-5efe-b3dd-24760f09c58b"
	originalID := fixtureconsts.Deployment1
	maskedID := GetMaskedDeploymentID(originalID, "test deployment")
	assert.Equal(t, expected, maskedID)
	assert.NotEqual(t, originalID, maskedID)
}
