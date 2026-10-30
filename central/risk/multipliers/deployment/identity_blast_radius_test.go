package deployment

import (
	"context"
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stretchr/testify/assert"
)

func TestIdentityBlastRadiusScore(t *testing.T) {
	multiplier := NewIdentityBlastRadius()

	deployment := &storage.Deployment{
		ServiceAccount:                "secret-reader",
		ServiceAccountPermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		AutomountServiceAccountToken:  true,
	}

	expectedScore := &storage.Risk_Result{
		Name: IdentityBlastRadiusHeading,
		Factors: []*storage.Risk_Result_Factor{
			{Message: "Service account \"secret-reader\" has cluster-admin permissions and its token is automounted"},
		},
		Score: 4,
	}
	protoassert.Equal(t, expectedScore, multiplier.Score(context.Background(), deployment, nil))
}

func TestIdentityBlastRadiusRequiresAutomountedToken(t *testing.T) {
	multiplier := NewIdentityBlastRadius()

	deployment := &storage.Deployment{
		ServiceAccount:                "secret-reader",
		ServiceAccountPermissionLevel: storage.PermissionLevel_CLUSTER_ADMIN,
		AutomountServiceAccountToken:  false,
	}

	assert.Nil(t, multiplier.Score(context.Background(), deployment, nil))
}
