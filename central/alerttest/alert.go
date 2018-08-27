package alerttest

import (
	"github.com/stackrox/rox/generated/api/v1"
)

// Constants for unit tests that need fake Alerts.
const (
	FakeAlertID     = "fake-alert-id"
	FakeClusterName = "fakeCluster"
	FakePolicyID    = "fake-policy-id"
)

// NewFakeListAlert constructs and returns a new V1.ListAlert object suitable for unit-testing.
func NewFakeListAlert() *v1.ListAlert {
	return &v1.ListAlert{
		Id: FakeAlertID,
		Policy: &v1.ListAlertPolicy{
			Id: FakePolicyID,
		},
		Deployment: &v1.ListAlertDeployment{
			ClusterName: FakeClusterName,
		},
	}
}

// NewFakeListAlertSlice constructs and returns a new slice of v1.ListAlert objects suitable for unit-testing.
func NewFakeListAlertSlice() []*v1.ListAlert {
	return []*v1.ListAlert{
		NewFakeListAlert(),
	}
}

// NewFakeAlert constructs and returns a new v1.Alert object suitable for unit-testing.
func NewFakeAlert() *v1.Alert {
	return &v1.Alert{
		Id: FakeAlertID,
	}
}
