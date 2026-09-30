package convert

import (
	"testing"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stretchr/testify/assert"
)

func TestAlertToListAlertSetsNodeResourceType(t *testing.T) {
	alert := &storage.Alert{
		Id:          "alert-1",
		ClusterId:   "cluster-1",
		ClusterName: "test-cluster",
		Entity: &storage.Alert_Node_{
			Node: &storage.Alert_Node{
				Id:   "node-1",
				Name: "test-node",
			},
		},
	}

	listAlert := AlertToListAlert(alert)

	assert.Equal(t, storage.ListAlert_NODE, listAlert.GetCommonEntityInfo().GetResourceType())
	assert.Equal(t, "test-cluster", listAlert.GetCommonEntityInfo().GetClusterName())
	assert.Equal(t, "cluster-1", listAlert.GetCommonEntityInfo().GetClusterId())
	assert.Equal(t, "test-node", listAlert.GetNode().GetName())
}

func TestAlertAndListAlertResourceTypesAreInSync(t *testing.T) {
	assert.Equal(t, storage.ListAlert_ResourceType_name[0], "DEPLOYMENT")
	assert.Equal(t, storage.Alert_Resource_ResourceType_name[0], "UNKNOWN")

	// ListAlert.ResourceType omits UNKNOWN but includes DEPLOYMENT and NODE,
	// so it has one more entry than Alert.Resource.ResourceType.
	listAlertOnlyTypes := 1
	assert.Equal(t, len(storage.Alert_Resource_ResourceType_value)+listAlertOnlyTypes, len(storage.ListAlert_ResourceType_value))
	for i, at := range storage.Alert_Resource_ResourceType_name {
		if r := storage.Alert_Resource_ResourceType(i); r == storage.Alert_Resource_UNKNOWN {
			continue
		}
		assert.Contains(t, storage.ListAlert_ResourceType_value, at)
		assert.Equal(t, at, storage.ListAlert_ResourceType_name[i])
	}
}

func TestToAlertResource(t *testing.T) {
	builtIn := ToAlertResource(&storage.KubernetesEvent{Object: &storage.KubernetesEvent_Object{
		Name:     "my-secret",
		Resource: storage.KubernetesEvent_Object_SECRETS,
	}}).Resource
	assert.Equal(t, storage.Alert_Resource_SECRETS, builtIn.GetResourceType())
	assert.Empty(t, builtIn.GetApiResource())

	custom := ToAlertResource(&storage.KubernetesEvent{Object: &storage.KubernetesEvent_Object{
		Name:        "my-app",
		Resource:    storage.KubernetesEvent_Object_UNKNOWN,
		ApiGroup:    "argoproj.io",
		ApiResource: "applications.argoproj.io",
	}}).Resource
	assert.Equal(t, storage.Alert_Resource_CUSTOM, custom.GetResourceType())
	assert.Equal(t, "applications.argoproj.io", custom.GetApiResource())

	listAlert := AlertToListAlert(&storage.Alert{Entity: &storage.Alert_Resource_{Resource: custom}})
	assert.Equal(t, storage.ListAlert_CUSTOM, listAlert.GetCommonEntityInfo().GetResourceType())
	assert.Equal(t, "applications.argoproj.io", listAlert.GetResource().GetApiResource())
}
