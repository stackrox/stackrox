package fixtures

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/images/types"
)

// GetAlert returns a Mock Alert
func GetAlert() *storage.Alert {
	return &storage.Alert{
		Id: "Alert1",
		Violations: []*storage.Alert_Violation{
			{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			},
			{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			},
		},
		Time:   ptypes.TimestampNow(),
		Policy: GetPolicy(),
		Deployment: &storage.Alert_Deployment{
			Name:        "nginx_server",
			Id:          "s79mdvmb6dsl",
			ClusterId:   "prod cluster",
			ClusterName: "prod cluster",
			Namespace:   "stackrox",
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
	}
}

// GetAlertWithID returns a mock alert with the specified id.
func GetAlertWithID(id string) *storage.Alert {
	alert := GetAlert()
	alert.Id = id
	return alert
}
