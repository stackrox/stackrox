package multipliers

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stackrox/rox/central/serviceaccount/datastore/mocks"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/assert"
)

func TestAutomountScore(t *testing.T) {
	cases := []struct {
		name           string
		serviceAccount *storage.ServiceAccount
		expected       *storage.Risk_Result
	}{
		{
			name: "Service Account with auto mount",
			serviceAccount: &storage.ServiceAccount{
				Name:           "service-account",
				AutomountToken: true,
				ClusterId:      "cluster",
				Namespace:      "namespace",
			},
			expected: &storage.Risk_Result{
				Name: RBACConfigurationHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Deployment is configured to automatically mount a token for service account \"service-account\""},
					{Message: "Service account \"service-account\" is configured to mount a token into the deployment by default"},
				},
				Score: 2,
			},
		},
		{
			name: "Service Account without auto mount",
			serviceAccount: &storage.ServiceAccount{
				Name:           "service-account",
				AutomountToken: false,
				ClusterId:      "cluster",
				Namespace:      "namespace",
			},
			expected: &storage.Risk_Result{
				Name: RBACConfigurationHeading,
				Factors: []*storage.Risk_Result_Factor{
					{Message: "Deployment is configured to automatically mount a token for service account \"service-account\""},
				},
				Score: 2,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			mockDatastore := mocks.NewMockDataStore(mockCtrl)
			q := search.NewQueryBuilder().
				AddExactMatches(search.ClusterID, c.serviceAccount.ClusterId).
				AddExactMatches(search.Namespace, c.serviceAccount.Namespace).
				AddExactMatches(search.ServiceAccountName, c.serviceAccount.Name).ProtoQuery()
			mockDatastore.EXPECT().SearchRawServiceAccounts(q).Return([]*storage.ServiceAccount{c.serviceAccount}, nil)

			mult := NewSecretAutomount(mockDatastore)
			deployment := getMockDeployment()
			result := mult.Score(deployment)
			assert.Equal(t, c.expected, result)
			mockCtrl.Finish()
		})
	}
}
