package metrics

import (
	"github.com/stackrox/rox/sensor/common/managedcentral"
)

func getHosting() string {
	if managedcentral.IsCentralManaged() {
		return "cloud-service"
	}
	return "self-managed"
}
