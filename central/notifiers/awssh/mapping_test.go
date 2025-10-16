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
	"google.golang.org/protobuf/proto"
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
	policy := &storage.Policy{}
	policy.SetId("policy-1")
	ar := &storage.Alert_Resource{}
	ar.SetName("secret1")
	ar.SetClusterId("cluster-1")
	ar.SetClusterName("cluster1")
	ar.SetNamespace("namespace1")
	ar.SetNamespaceId("namespace-1")
	ar.SetResourceType(storage.Alert_Resource_SECRETS)
	testAuditLogAlert := &storage.Alert{}
	testAuditLogAlert.SetId("audit-1")
	testAuditLogAlert.SetPolicy(policy)
	testAuditLogAlert.SetResource(proto.ValueOrDefault(ar))

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
	policy := &storage.Policy{}
	policy.SetId("policy-1")
	ad := &storage.Alert_Deployment{}
	ad.SetName("deployment1")
	ad.SetClusterId("cluster-1")
	ad.SetClusterName("cluster1")
	ad.SetNamespace("namespace1")
	ad.SetNamespaceId("namespace-1")
	testDeploymentAlert := &storage.Alert{}
	testDeploymentAlert.SetId("audit-1")
	testDeploymentAlert.SetPolicy(policy)
	testDeploymentAlert.SetDeployment(proto.ValueOrDefault(ad))
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
