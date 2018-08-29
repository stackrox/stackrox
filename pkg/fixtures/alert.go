package fixtures

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/api/v1"
)

// GetAlert returns a Mock Alert
func GetAlert() *v1.Alert {
	return &v1.Alert{
		Id: "Alert1",
		Violations: []*v1.Alert_Violation{
			{
				Message: "Deployment is affected by 'CVE-2017-15804'",
			},
			{
				Message: "Deployment is affected by 'CVE-2017-15670'",
			},
		},
		Time:       ptypes.TimestampNow(),
		Policy:     GetPolicy(),
		Deployment: GetDeployment(),
	}
}

// GetAlertWithID returns a mock alert with the specified id.
func GetAlertWithID(id string) *v1.Alert {
	alert := GetAlert()
	alert.Id = id
	return alert
}
