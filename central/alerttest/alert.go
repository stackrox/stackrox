package alerttest

import (
	"github.com/stackrox/rox/generated/storage"
)

// Constants for unit tests that need fake Alerts.
const (
	FakeAlertID             = "fake-alert-id"
	FakeClusterName         = "fakeCluster"
	FakePolicyID            = "fake-policy-id"
	FakeCommentID           = "fake-comment-id"
	FakeAlertCommentMessage = "fake-alert-comment-message"
	FakeTag1                = "FakeTag1"
	FakeTag2                = "FakeTag2"
	FakeTag3                = "FakeTag3"
)

// NewFakeListAlert constructs and returns a new V1.ListAlert object suitable for unit-testing.
func NewFakeListAlert() *storage.ListAlert {
	return &storage.ListAlert{
		Id: FakeAlertID,
		Policy: &storage.ListAlertPolicy{
			Id: FakePolicyID,
		},
		Deployment: &storage.ListAlertDeployment{
			ClusterName: FakeClusterName,
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
		Tags:           NewFakeTwoTags(),
	}
}

// NewFakeAlertWithThreeTags constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithThreeTags() *storage.Alert {
	return &storage.Alert{
		Id:             FakeAlertID,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Tags:           NewFakeThreeTags(),
	}
}

// NewFakeAlertWithOneTag constructs and returns a new storage.Alert object(with tags) suitable for unit-testing.
func NewFakeAlertWithOneTag() *storage.Alert {
	return &storage.Alert{
		Id:             FakeAlertID,
		LifecycleStage: storage.LifecycleStage_RUNTIME,
		Tags:           []string{FakeTag3},
	}
}

// NewFakeAlertComment constructs and returns a new storage.Comment object suitable for unit-testing.
func NewFakeAlertComment() *storage.Comment {
	return &storage.Comment{
		ResourceId:     FakeAlertID,
		CommentId:      FakeCommentID,
		CommentMessage: FakeAlertCommentMessage,
	}
}

//NewFakeTwoTags constructs and returns a new slice with two fake tags
func NewFakeTwoTags() []string {
	return []string{FakeTag1, FakeTag2}
}

//NewFakeTwoTagsHasOverlap constructs and returns a new slice with two fake tags has overlap with slice constructed by NewFakeTwoTags()
func NewFakeTwoTagsHasOverlap() []string {
	return []string{FakeTag2, FakeTag3}
}

//NewFakeThreeTags constructs and returns a new slice with three fake tags
func NewFakeThreeTags() []string {
	return []string{FakeTag1, FakeTag2, FakeTag3}
}
