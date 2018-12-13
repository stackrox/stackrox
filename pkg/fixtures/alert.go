package fixtures

import (
	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
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
		Time:       ptypes.TimestampNow(),
		Policy:     GetPolicy(),
		Deployment: GetDeployment(),
	}
}

// GetAlertWithID returns a mock alert with the specified id.
func GetAlertWithID(id string) *storage.Alert {
	alert := GetAlert()
	alert.Id = id
	return alert
}
