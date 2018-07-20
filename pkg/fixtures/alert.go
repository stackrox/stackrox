package fixtures

import (
	"bitbucket.org/stack-rox/apollo/generated/api/v1"
	ptypes "github.com/gogo/protobuf/types"
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
