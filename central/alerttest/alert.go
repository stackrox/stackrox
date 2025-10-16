package alerttest

import (
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures/fixtureconsts"
	"google.golang.org/protobuf/proto"
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
	lap := &storage.ListAlertPolicy{}
	lap.SetId(FakePolicyID)
	lc := &storage.ListAlert_CommonEntityInfo{}
	lc.SetClusterName(FakeClusterName)
	lad := &storage.ListAlertDeployment{}
	lad.SetClusterName(FakeClusterName)
	listAlert := &storage.ListAlert{}
	listAlert.SetId(FakeAlertID)
	listAlert.SetPolicy(lap)
	listAlert.SetCommonEntityInfo(lc)
	listAlert.SetDeployment(proto.ValueOrDefault(lad))
	return listAlert
}

// NewFakeListAlertSlice constructs and returns a new slice of storage.ListAlert objects suitable for unit-testing.
func NewFakeListAlertSlice() []*storage.ListAlert {
	return []*storage.ListAlert{
		NewFakeListAlert(),
	}
}

// NewFakeAlert constructs and returns a new storage.Alert object suitable for unit-testing.
func NewFakeAlert() *storage.Alert {
	alert := &storage.Alert{}
	alert.SetId(FakeAlertID)
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	return alert
}

// NewFakeAlertWithTwoTags constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithTwoTags() *storage.Alert {
	alert := &storage.Alert{}
	alert.SetId(FakeAlertID)
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	return alert
}

// NewFakeAlertWithThreeTags constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithThreeTags() *storage.Alert {
	alert := &storage.Alert{}
	alert.SetId(FakeAlertID)
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	return alert
}

// NewFakeAlertWithOneTag constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithOneTag() *storage.Alert {
	alert := &storage.Alert{}
	alert.SetId(FakeAlertID)
	alert.SetLifecycleStage(storage.LifecycleStage_RUNTIME)
	return alert
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
