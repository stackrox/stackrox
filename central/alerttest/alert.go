package alerttest

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
)

// Constants for unit tests that need fake Alerts.
const (
	FakeAlertID     = fixtureconsts.AlertFake
	FakeClusterName = "fakeCluster"
	FakePolicyID    = fixtureconsts.PolicyFake
	FakeTag1        = "FakeTag1"
	FakeTag2        = "FakeTag2"
	FakeTag3        = "FakeTag3"
)

// NewFakeListAlert constructs and returns a new V1.ListAlert object suitable for unit-testing.
func NewFakeListAlert() *storage.ListAlert {
	return &storage.ListAlert{
		Id: FakeAlertID,
		Policy: &storage.ListAlertPolicy{
			Id: FakePolicyID,
		},
		CommonEntityInfo: &storage.ListAlert_CommonEntityInfo{
			ClusterName: FakeClusterName,
		},
		Entity: &storage.ListAlert_Deployment{
			Deployment: &storage.ListAlertDeployment{
				ClusterName: FakeClusterName,
			},
		},
	}
}

// NewFakeListAlertSlice constructs and returns a new slice of storage.ListAlert objects suitable for unit-testing.
func NewFakeListAlertSlice() []*storage.ListAlert {
	return []*storage.ListAlert{
		NewFakeListAlert(),
	}
}

// NewFakeAlert constructs and returns a new storage.Alert object suitable for unit-testing.
func NewFakeAlert() *storage.Alert {
	return &storage.Alert{
		Id:             FakeAlertID,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}
}

// NewFakeAlertWithTwoTags constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithTwoTags() *storage.Alert {
	return &storage.Alert{
		Id:             FakeAlertID,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}
}

// NewFakeAlertWithThreeTags constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithThreeTags() *storage.Alert {
	return &storage.Alert{
		Id:             FakeAlertID,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}
}

// NewFakeAlertWithOneTag constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithOneTag() *storage.Alert {
	return &storage.Alert{
		Id:             FakeAlertID,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
	}
}

// NewFakeTwoTags constructs and returns a new slice with two fake tags
func NewFakeTwoTags() []string {
	return []string{FakeTag1, FakeTag2}
}

// NewFakeTwoTagsHasOverlap constructs and returns a new slice with two fake tags has overlap with slice constructed by NewFakeTwoTags()
func NewFakeTwoTagsHasOverlap() []string {
	return []string{FakeTag2, FakeTag3}
}

// NewFakeThreeTags constructs and returns a new slice with three fake tags
func NewFakeThreeTags() []string {
	return []string{FakeTag1, FakeTag2, FakeTag3}
}
