package awssh

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	securityhubTypes "github.com/aws/aws-sdk-go-v2/service/securityhub/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/defaults/policies"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllDefaultCategoriesHaveMappings(t *testing.T) {
	defaultPolicies, err := policies.DefaultPolicies()
	require.NoError(t, err)

	categoryMapSet := set.NewStringSet()
	for k := range categoryMap {
		categoryMapSet.Add(k)
	}

	for _, policy := range defaultPolicies {
		for _, category := range policy.GetCategories() {
			_, ok := categoryMap[strings.ToLower(category)]
			if ok {
				categoryMapSet.Remove(strings.ToLower(category))
			}
			assert.True(t, ok, "category %s not mapped", category)
		}
	}
	// Ensure that all categories in the map are used in policies.
	assert.Len(t, categoryMapSet, 0)
}

func TestGetEntitySectionResourceAlert(t *testing.T) {
	testAuditLogAlert := storage.Alert{
		Id: "audit-1",
		Policy: &storage.Policy{
			Id: "policy-1",
		},
		Entity: &storage.Alert_Resource_{
			Resource: &storage.Alert_Resource{
				Name:         "secret1",
				ClusterId:    "cluster-1",
				ClusterName:  "cluster1",
				Namespace:    "namespace1",
				NamespaceId:  "namespace-1",
				ResourceType: storage.Alert_Resource_SECRETS,
			},
		},
	}

	expectedResource := securityhubTypes.Resource{
		Id:   aws.String("resource: secret1"),
		Type: aws.String(resourceTypeOther),
		Details: &securityhubTypes.ResourceDetails{
			Other: map[string]string{
				"cluster-name":       "cluster1",
				"resource-name":      "secret1",
				"resource-namespace": "namespace1",
				"resource-type":      "SECRETS",
			},
		},
	}

	resources := getEntitySection(&testAuditLogAlert)
	assert.NotNil(t, resources)
	assert.Len(t, resources, 1)
	assert.NotNil(t, resources[0].Details)
	assert.Equal(t, expectedResource, resources[0])
}

func TestGetEntitySectionDeploymentAlert(t *testing.T) {
	testDeploymentAlert := storage.Alert{
		Id: "audit-1",
		Policy: &storage.Policy{
			Id: "policy-1",
		},
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
				Name:        "deployment1",
				ClusterId:   "cluster-1",
				ClusterName: "cluster1",
				Namespace:   "namespace1",
				NamespaceId: "namespace-1",
			},
		},
	}
	expectedResource := securityhubTypes.Resource{
		Id:   aws.String("deployment: deployment1"),
		Type: aws.String(resourceTypeOther),
		Details: &securityhubTypes.ResourceDetails{
			Other: map[string]string{
				"cluster-name":         "cluster1",
				"deployment-name":      "deployment1",
				"deployment-namespace": "namespace1",
			},
		},
	}
	resources := getEntitySection(&testDeploymentAlert)
	assert.NotNil(t, resources)
	assert.Len(t, resources, 1)
	assert.NotNil(t, resources[0].Details)
	assert.Equal(t, expectedResource, resources[0])
}
