package fixtures

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
	"google.golang.org/protobuf/proto"
)

func copyScopingInfo(alert *storage.Alert) *storage.Alert {
	switch alert.WhichEntity() {
	case storage.Alert_Deployment_case:
		alert.SetClusterName(alert.GetDeployment().GetClusterName())
		alert.SetClusterId(alert.GetDeployment().GetClusterId())
		alert.SetNamespace(alert.GetDeployment().GetNamespace())
		alert.SetNamespaceId(alert.GetDeployment().GetNamespaceId())
	case storage.Alert_Resource_case:
		alert.SetClusterName(alert.GetResource().GetClusterName())
		alert.SetClusterId(alert.GetResource().GetClusterId())
		alert.SetNamespace(alert.GetResource().GetNamespace())
		alert.SetNamespaceId(alert.GetResource().GetNamespaceId())
	}
	return alert
}

// GetScopedDeploymentAlert returns a Mock alert attached to a deployment belonging to the input scope
func GetScopedDeploymentAlert(ID string, clusterID string, namespace string) *storage.Alert {
	return copyScopingInfo(storage.Alert_builder{
		Id: ID,
		Violations: []*storage.Alert_Violation{
			storage.Alert_Violation_builder{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			}.Build(),
			storage.Alert_Violation_builder{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			}.Build(),
			storage.Alert_Violation_builder{
				Message: "This is a kube event violation",
				KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "pod", Value: "nginx"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "container", Value: "nginx"}.Build(),
					},
				}.Build(),
			}.Build(),
		},
		ProcessViolation: storage.Alert_ProcessViolation_builder{
			Message: "This is a process violation",
		}.Build(),
		Time:   protocompat.TimestampNow(),
		Policy: GetPolicy(),
		Deployment: storage.Alert_Deployment_builder{
			Name:        "nginx_server",
			Id:          fixtureconsts.Deployment1,
			ClusterId:   clusterID,
			ClusterName: "prod cluster",
			Namespace:   namespace,
			Labels: map[string]string{
				"com.docker.stack.namespace":    "prevent",
				"com.docker.swarm.service.name": "prevent_sensor",
				"email":                         "vv@stackrox.com",
				"owner":                         "stackrox",
			},
			Containers: []*storage.Alert_Deployment_Container{
				storage.Alert_Deployment_Container_builder{
					Name:  "nginx110container",
					Image: types.ToContainerImage(LightweightDeploymentImage()),
				}.Build(),
			},
		}.Build(),
	}.Build())
}

// GetAlert returns a Mock Alert
func GetAlert() *storage.Alert {
	return GetScopedDeploymentAlert(fixtureconsts.Alert1, fixtureconsts.Cluster1, "stackrox")
}

// GetAlertWithMitre returns a mock Alert with MITRE ATT&CK
func GetAlertWithMitre() *storage.Alert {
	alert := GetAlert()
	alert.SetPolicy(GetPolicyWithMitre())
	return alert
}

// GetResourceAlert returns a Mock Alert with a resource entity
func GetResourceAlert() *storage.Alert {
	return GetScopedResourceAlert(fixtureconsts.Alert1, fixtureconsts.Cluster1, "stackrox")
}

// GetScopedResourceAlert returns a Mock alert with a resource entity belonging to the input scope
func GetScopedResourceAlert(ID string, clusterID string, namespace string) *storage.Alert {
	return copyScopingInfo(storage.Alert_builder{
		Id: ID,
		Violations: []*storage.Alert_Violation{
			storage.Alert_Violation_builder{
				Message: "Access to secret \"my-secret\" in \"cluster-id / stackrox\"",
				Type:    storage.Alert_Violation_K8S_EVENT,
				KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "Kubernetes API Verb", Value: "CREATE"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "username", Value: "test-user"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "user groups", Value: "groupA, groupB"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "resource", Value: "/api/v1/namespace/stackrox/secrets/my-secret"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "user agent", Value: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "IP address", Value: "192.168.0.1, 127.0.0.1"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "impersonated username", Value: "central-service-account"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "impersonated user groups", Value: "service-accounts, groupB"}.Build(),
					},
				}.Build(),
			}.Build(),
		},
		Time:   protocompat.TimestampNow(),
		Policy: GetAuditLogEventSourcePolicy(),
		Resource: storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_SECRETS,
			Name:         "my-secret",
			ClusterId:    clusterID, // "cluster-id",
			ClusterName:  "prod cluster",
			Namespace:    namespace, // "stackrox",
			NamespaceId:  fixtureconsts.Namespace1,
		}.Build(),
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}.Build())
}

// GetClusterResourceAlert returns a Mock Alert with a resource entity that is cluster wide (i.e. has no namespace)
func GetClusterResourceAlert() *storage.Alert {
	policy := GetAuditLogEventSourcePolicy()
	policy.GetPolicySections()[0].GetPolicyGroups()[0].GetValues()[0].SetValue("CLUSTER_ROLES")

	return copyScopingInfo(storage.Alert_builder{
		Id: fixtureconsts.Alert1,
		Violations: []*storage.Alert_Violation{
			storage.Alert_Violation_builder{
				Message: "Access to cluster role \"my-cluster-role\"",
				Type:    storage.Alert_Violation_K8S_EVENT,
				KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "Kubernetes API Verb", Value: "CREATE"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "username", Value: "test-user"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "user groups", Value: "groupA, groupB"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "resource", Value: "/apis/rbac.authorization.k8s.io/v1/clusterroles/my-cluster-role"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "user agent", Value: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "IP address", Value: "192.168.0.1, 127.0.0.1"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "impersonated username", Value: "central-service-account"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "impersonated user groups", Value: "service-accounts, groupB"}.Build(),
					},
				}.Build(),
			}.Build(),
		},
		Time:   protocompat.TimestampNow(),
		Policy: policy,
		Resource: storage.Alert_Resource_builder{
			ResourceType: storage.Alert_Resource_CLUSTER_ROLES,
			Name:         "my-cluster-role",
			ClusterId:    fixtureconsts.Cluster3,
			ClusterName:  "prod cluster",
		}.Build(),
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}.Build())
}

// GetImageAlert returns a Mock alert with an image for entity
func GetImageAlert() *storage.Alert {
	return getImageAlertWithID("Alert1")
}

func getImageAlertWithID(ID string) *storage.Alert {
	imageAlert := GetAlertWithID(ID)
	imageAlert.SetImage(proto.ValueOrDefault(types.ToContainerImage(GetImage())))

	return imageAlert
}

// GetNetworkAlert returns a Mock Alert with a network flow violations
func GetNetworkAlert() *storage.Alert {
	return copyScopingInfo(storage.Alert_builder{
		Id:             fixtureconsts.Alert1,
		Policy:         GetNetworkFlowPolicy(),
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Deployment: storage.Alert_Deployment_builder{
			Id:          fixtureconsts.Deployment1,
			Name:        "central",
			Type:        "Deployment",
			Namespace:   "stackrox",
			NamespaceId: fixtureconsts.Namespace1,
			Labels: map[string]string{
				"app":                         "central",
				"app.kubernetes.io/component": "central",
			},
			ClusterId:   fixtureconsts.Cluster1,
			ClusterName: "remote",
			Containers: []*storage.Alert_Deployment_Container{storage.Alert_Deployment_Container_builder{
				Name:  "some-container",
				Image: types.ToContainerImage(LightweightDeploymentImage()),
			}.Build()},
			Annotations: map[string]string{
				"email":                     "support@stackrox.com",
				"meta.helm.sh/release-name": "stackrox-central-services",
			},
		}.Build(),
		Violations: []*storage.Alert_Violation{
			storage.Alert_Violation_builder{
				Message: "Unexpected network flow found in deployment. Source name: 'central'. Destination name: 'External Entities'. Destination port: '9'. Protocol: 'L4_PROTOCOL_UDP'.",
				NetworkFlowInfo: storage.Alert_Violation_NetworkFlowInfo_builder{
					Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
					Source: storage.Alert_Violation_NetworkFlowInfo_Entity_builder{
						Name:                "central",
						EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
						DeploymentNamespace: "stackrox",
						DeploymentType:      "Deployment",
					}.Build(),
					Destination: storage.Alert_Violation_NetworkFlowInfo_Entity_builder{
						Name:                "External Entities",
						EntityType:          storage.NetworkEntityInfo_INTERNET,
						DeploymentNamespace: "internet",
						Port:                9,
					}.Build(),
				}.Build(),
				Type: storage.Alert_Violation_NETWORK_FLOW,
				Time: protocompat.TimestampNow(),
			}.Build(),
			storage.Alert_Violation_builder{
				Message: "Unexpected network flow found in deployment. Source name: 'central'. Destination name: 'scanner'. Destination port: '8080'. Protocol: 'L4_PROTOCOL_TCP'.",
				NetworkFlowInfo: storage.Alert_Violation_NetworkFlowInfo_builder{
					Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
					Source: storage.Alert_Violation_NetworkFlowInfo_Entity_builder{
						Name:                "central",
						EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
						DeploymentNamespace: "stackrox",
						DeploymentType:      "Deployment",
					}.Build(),
					Destination: storage.Alert_Violation_NetworkFlowInfo_Entity_builder{
						Name:                "scanner",
						EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
						DeploymentNamespace: "stackrox",
						DeploymentType:      "Deployment",
						Port:                8080,
					}.Build(),
				}.Build(),
				Type: storage.Alert_Violation_NETWORK_FLOW,
				Time: protocompat.TimestampNow(),
			}.Build(),
		},
		Time:          protocompat.TimestampNow(),
		FirstOccurred: protocompat.TimestampNow(),
	}.Build())
}

// GetAlertWithID returns a mock alert with the specified id.
func GetAlertWithID(id string) *storage.Alert {
	alert := GetAlert()
	alert.SetId(id)
	return alert
}

// GetSACTestAlertSet returns a set of mock alerts that can be used for scoped access control tests
func GetSACTestAlertSet() []*storage.Alert {
	alerts := make([]*storage.Alert, 0, 19)
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceA))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster1, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceB))
	alerts = append(alerts, GetScopedDeploymentAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceC))
	alerts = append(alerts, GetScopedResourceAlert(uuid.NewV4().String(), testconsts.Cluster2, testconsts.NamespaceC))
	alerts = append(alerts, getImageAlertWithID(uuid.NewV4().String()))
	return alerts
}

// GetSerializationTestAlert returns a mock alert that can be used
// for serialization and deserialization tests.
//
// The equivalent JSON serialized format can be obtained with
// GetJSONSerializedTestAlert.
func GetSerializationTestAlert() *storage.Alert {
	return storage.Alert_builder{
		Id: fixtureconsts.Alert1,
		Violations: []*storage.Alert_Violation{
			storage.Alert_Violation_builder{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			}.Build(),
			storage.Alert_Violation_builder{
				Message: "This is a kube event violation",
				KeyValueAttrs: storage.Alert_Violation_KeyValueAttrs_builder{
					Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "pod", Value: "nginx"}.Build(),
						storage.Alert_Violation_KeyValueAttrs_KeyValueAttr_builder{Key: "container", Value: "nginx"}.Build(),
					},
				}.Build(),
			}.Build(),
		},
		ProcessViolation: storage.Alert_ProcessViolation_builder{
			Message: "This is a process violation",
		}.Build(),
		ClusterId:   fixtureconsts.Cluster1,
		ClusterName: "prod cluster",
		Namespace:   "stackrox",
	}.Build()
}

// GetJSONSerializedTestAlertWithDefaults returns the ProtoJSON serialized form
// of the alert returned by GetSerializationTestAlert,
// with default value emission during serialization.
func GetJSONSerializedTestAlertWithDefaults() string {
	return `{
	"id": "aeaaaaaa-bbbb-4011-0000-111111111111",
	"clusterId": "caaaaaaa-bbbb-4011-0000-111111111111",
	"clusterName": "prod cluster",
	"enforcement": null,
	"entityType": "UNSET",
	"firstOccurred": null,
	"lifecycleStage": "DEPLOY",
	"namespace": "stackrox",
	"namespaceId": "",
	"platformComponent": false,
	"policy": null,
	"processViolation": {
		"message": "This is a process violation",
		"processes": []
	},
	"resolvedAt":null,
	"state": "ACTIVE",
	"time": null,
	"violations": [
		{
			"message": "Deployment is affected by 'CVE-2017-15670'",
			"time": null,
			"type": "GENERIC"
		},
		{
			"message": "This is a kube event violation",
			"keyValueAttrs": {
				"attrs": [
					{"key": "pod", "value": "nginx"},
					{"key": "container", "value": "nginx"}
				]
			},
			"time": null,
			"type": "GENERIC"
		}
	]
}`
}

// GetJSONSerializedTestAlert returns the ProtoJSON serialized form
// of the alert returned by GetSerializationTestAlert.
func GetJSONSerializedTestAlert() string {
	return `{
	"id": "aeaaaaaa-bbbb-4011-0000-111111111111",
	"clusterId": "caaaaaaa-bbbb-4011-0000-111111111111",
	"clusterName": "prod cluster",
	"namespace": "stackrox",
	"processViolation": {
		"message": "This is a process violation"
	},
	"violations": [
		{
			"message": "Deployment is affected by 'CVE-2017-15670'"
		},
		{
			"message": "This is a kube event violation",
			"keyValueAttrs": {
				"attrs": [
					{"key": "pod", "value": "nginx"},
					{"key": "container", "value": "nginx"}
				]
			}
		}
	]
}`
}
