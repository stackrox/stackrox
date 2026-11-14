package accesscontrol

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDefaultPermissionSet(t *testing.T) {
	for ps := range DefaultPermissionSetIDs {
		assert.True(t, IsDefaultPermissionSet(ps), ps)
	}
}
