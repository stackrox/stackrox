package fixtures

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"github.com/stackrox/rox/pkg/images/types"
	"github.com/stackrox/rox/pkg/sac/testconsts"
	"github.com/stackrox/rox/pkg/uuid"
)

func copyScopingInfo(alert *storage.Alert) *storage.Alert {
	switch entity := alert.Entity.(type) {
	case *storage.Alert_Deployment_:
		alert.ClusterName = entity.Deployment.ClusterName
		alert.ClusterId = entity.Deployment.ClusterId
		alert.Namespace = entity.Deployment.Namespace
		alert.NamespaceId = entity.Deployment.NamespaceId
	case *storage.Alert_Resource_:
		alert.ClusterName = entity.Resource.ClusterName
		alert.ClusterId = entity.Resource.ClusterId
		alert.Namespace = entity.Resource.Namespace
		alert.NamespaceId = entity.Resource.NamespaceId
	}
	return alert
}

// GetScopedDeploymentAlert returns a Mock alert attached to a deployment belonging to the input scope
func GetScopedDeploymentAlert(ID string, clusterID string, namespace string) *storage.Alert {
	return copyScopingInfo(&storage.Alert{
		Id: ID,
		Violations: []*storage.Alert_Violation{
			{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			},
			{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			},
			{
				Message: "This is a kube event violation",
				MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
					KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
						Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
							{Key: "pod", Value: "nginx"},
							{Key: "container", Value: "nginx"},
						},
					},
				},
			},
		},
		ProcessViolation: &storage.Alert_ProcessViolation{
			Message: "This is a process violation",
		},
		Time:   ptypes.TimestampNow(),
		Policy: GetPolicy(),
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
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
					{
						Name:  "nginx110container",
						Image: types.ToContainerImage(LightweightDeploymentImage()),
					},
				},
			},
		},
	})
}

// GetAlert returns a Mock Alert
func GetAlert() *storage.Alert {
	return GetScopedDeploymentAlert(fixtureconsts.Alert1, fixtureconsts.Cluster1, "stackrox")
}

// GetAlertWithMitre returns a mock Alert with MITRE ATT&CK
func GetAlertWithMitre() *storage.Alert {
	alert := GetAlert()
	alert.Policy = GetPolicyWithMitre()
	return alert
}

// GetResourceAlert returns a Mock Alert with a resource entity
func GetResourceAlert() *storage.Alert {
	return GetScopedResourceAlert(fixtureconsts.Alert1, fixtureconsts.Cluster1, "stackrox")
}

// GetScopedResourceAlert returns a Mock alert with a resource entity belonging to the input scope
func GetScopedResourceAlert(ID string, clusterID string, namespace string) *storage.Alert {
	return copyScopingInfo(&storage.Alert{
		Id: ID,
		Violations: []*storage.Alert_Violation{
			{
				Message: "Access to secret \"my-secret\" in \"cluster-id / stackrox\"",
				Type:    storage.Alert_Violation_K8S_EVENT,
				MessageAttributes: &storage.Alert_Violation_KeyValueAttrs_{
					KeyValueAttrs: &storage.Alert_Violation_KeyValueAttrs{
						Attrs: []*storage.Alert_Violation_KeyValueAttrs_KeyValueAttr{
							{Key: "Kubernetes API Verb", Value: "CREATE"},
							{Key: "username", Value: "test-user"},
							{Key: "user groups", Value: "groupA, groupB"},
							{Key: "resource", Value: "/api/v1/namespace/stackrox/secrets/my-secret"},
							{Key: "user agent", Value: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f"},
							{Key: "IP address", Value: "192.168.0.1, 127.0.0.1"},
							{Key: "impersonated username", Value: "central-service-account"},
							{Key: "impersonated user groups", Value: "service-accounts, groupB"},
						},
					},
				},
			},
		},
		Time:   ptypes.TimestampNow(),
		Policy: GetAuditLogEventSourcePolicy(),
		Entity: &storage.Alert_Resource_{
			Resource: &storage.Alert_Resource{
				ResourceType: storage.Alert_Resource_SECRETS,
				Name:         "my-secret",
				ClusterId:    clusterID, // "cluster-id",
				ClusterName:  "prod cluster",
				Namespace:    namespace, // "stackrox",
				NamespaceId:  fixtureconsts.Namespace1,
			},
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	})
}

// GetImageAlert returns a Mock alert with an image for entity
func GetImageAlert() *storage.Alert {
	return getImageAlertWithID("Alert1")
}

func getImageAlertWithID(ID string) *storage.Alert {
	imageAlert := GetAlertWithID(ID)
	imageAlert.Entity = &storage.Alert_Image{Image: types.ToContainerImage(GetImage())}

	return imageAlert
}

// GetNetworkAlert returns a Mock Alert with a network flow violations
func GetNetworkAlert() *storage.Alert {
	return copyScopingInfo(&storage.Alert{
		Id:             fixtureconsts.Alert1,
		Policy:         GetNetworkFlowPolicy(),
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Entity: &storage.Alert_Deployment_{
			Deployment: &storage.Alert_Deployment{
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
				Containers: []*storage.Alert_Deployment_Container{{
					Name:  "some-container",
					Image: types.ToContainerImage(LightweightDeploymentImage()),
				}},
				Annotations: map[string]string{
					"email":                     "support@stackrox.com",
					"meta.helm.sh/release-name": "stackrox-central-services",
				},
			},
		},
		Violations: []*storage.Alert_Violation{
			{
				Message: "Unexpected network flow found in deployment. Source name: 'central'. Destination name: 'External Entities'. Destination port: '9'. Protocol: 'L4_PROTOCOL_UDP'.",
				MessageAttributes: &storage.Alert_Violation_NetworkFlowInfo_{
					NetworkFlowInfo: &storage.Alert_Violation_NetworkFlowInfo{
						Protocol: storage.L4Protocol_L4_PROTOCOL_UDP,
						Source: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "central",
							EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
							DeploymentNamespace: "stackrox",
							DeploymentType:      "Deployment",
						},
						Destination: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "External Entities",
							EntityType:          storage.NetworkEntityInfo_INTERNET,
							DeploymentNamespace: "internet",
							Port:                9,
						},
					},
				},
				Type: storage.Alert_Violation_NETWORK_FLOW,
				Time: ptypes.TimestampNow(),
			},
			{
				Message: "Unexpected network flow found in deployment. Source name: 'central'. Destination name: 'scanner'. Destination port: '8080'. Protocol: 'L4_PROTOCOL_TCP'.",
				MessageAttributes: &storage.Alert_Violation_NetworkFlowInfo_{
					NetworkFlowInfo: &storage.Alert_Violation_NetworkFlowInfo{
						Protocol: storage.L4Protocol_L4_PROTOCOL_TCP,
						Source: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "central",
							EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
							DeploymentNamespace: "stackrox",
							DeploymentType:      "Deployment",
						},
						Destination: &storage.Alert_Violation_NetworkFlowInfo_Entity{
							Name:                "scanner",
							EntityType:          storage.NetworkEntityInfo_DEPLOYMENT,
							DeploymentNamespace: "stackrox",
							DeploymentType:      "Deployment",
							Port:                8080,
						},
					},
				},
				Type: storage.Alert_Violation_NETWORK_FLOW,
				Time: ptypes.TimestampNow(),
			},
		},
		Time:          ptypes.TimestampNow(),
		FirstOccurred: ptypes.TimestampNow(),
	})
}

// GetAlertWithID returns a mock alert with the specified id.
func GetAlertWithID(id string) *storage.Alert {
	alert := GetAlert()
	alert.Id = id
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
