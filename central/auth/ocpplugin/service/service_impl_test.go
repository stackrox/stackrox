package service

import (
	"testing"

	"github.com/stackrox/rox/pkg/defaults/accesscontrol"
	"github.com/stretchr/testify/assert"
)

func TestGetPermissionSet(t *testing.T) {
	analystID := accesscontrol.DefaultPermissionSetIDs[accesscontrol.Analyst]
	assert.Equal(t, analystID, getPermissionSet(t.Context(), nil).GetId())
}
