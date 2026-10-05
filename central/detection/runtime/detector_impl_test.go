package runtime

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDeploymentInactive(t *testing.T) {
	tests := map[string]struct {
		deployment *storage.Deployment
		exists     bool
		err        error
		inactive   bool
	}{
		"present deployment": {
			deployment: &storage.Deployment{Id: "deployment-id"},
			exists:     true,
			inactive:   false,
		},
		"missing deployment": {
			inactive: true,
		},
		"datastore error": {
			err: errors.New("database unavailable"),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			deployments := mocks.NewMockDataStore(gomock.NewController(t))
			deployments.EXPECT().GetDeployment(gomock.Any(), "deployment-id").Return(test.deployment, test.exists, test.err)

			detector := NewDetector(nil, deployments)
			assert.Equal(t, test.inactive, detector.DeploymentInactive("deployment-id"))
		})
	}
}
