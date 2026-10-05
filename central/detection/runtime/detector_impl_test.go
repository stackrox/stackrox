package runtime

import (
	"errors"
	"testing"

	"github.com/stackrox/rox/central/deployment/datastore/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestDeploymentInactive(t *testing.T) {
	tests := map[string]struct {
		exists   bool
		err      error
		inactive bool
	}{
		"present deployment": {
			exists:   true,
			inactive: false,
		},
		"missing deployment": {
			exists:   false,
			inactive: true,
		},
		"datastore error": {
			err:      errors.New("database unavailable"),
			inactive: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			deployments := mocks.NewMockDataStore(gomock.NewController(t))
			deployments.EXPECT().DeploymentExists(gomock.Any(), "deployment-id").Return(test.exists, test.err)

			detector := NewDetector(nil, deployments)
			assert.Equal(t, test.inactive, detector.DeploymentInactive("deployment-id"))
		})
	}
}
