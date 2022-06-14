package fixtures

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/stackrox/generated/storage"
	"github.com/stackrox/stackrox/pkg/images/types"
	"github.com/stackrox/stackrox/pkg/sac/testconsts"
	"github.com/stackrox/stackrox/pkg/uuid"
)

// GetScopedDeploymentAlert returns a Mock alert attached to a deployment belonging to the input scope
func GetScopedDeploymentAlert(ID string, clusterID string, namespace string) *storage.Alert {
	return &storage.Alert{
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
				Id:          "s79mdvmb6dsl",
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
	}
}

// GetAlert returns a Mock Alert
func GetAlert() *storage.Alert {
	return GetScopedDeploymentAlert("Alert1", "prod cluster", "stackrox")
}

// GetAlertWithMitre returns a mock Alert with MITRE ATT&CK
func GetAlertWithMitre() *storage.Alert {
	alert := GetAlert()
	alert.Policy = GetPolicyWithMitre()
	return alert
}

// GetResourceAlert returns a Mock Alert with a resource entity
func GetResourceAlert() *storage.Alert {
	return GetScopedResourceAlert("some-resource-alert-on-secret", "cluster-id", "stackrox")
}

// GetScopedResourceAlert returns a Mock alert with a resource entity belonging to the input scope
func GetScopedResourceAlert(ID string, clusterID string, namespace string) *storage.Alert {
	return &storage.Alert{
		Id: ID, // "some-resource-alert-on-secret",
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
				NamespaceId:  "aaaa-bbbb-cccc-dddd",
			},
		},
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}
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
