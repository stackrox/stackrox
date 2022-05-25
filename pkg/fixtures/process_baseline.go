package fixtures

import (
	"time"

	ptypes "github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/uuid"
)

// Test fixtures for tests involving excluded scopes.

// GetProcessBaseline returns an empty process baseline
// with a random container name and deployment ID.
func GetProcessBaseline() *storage.ProcessBaseline {
	createStamp, _ := ptypes.TimestampProto(time.Now())
	processName := uuid.NewV4().String()
	process := &storage.BaselineElement{
		Element: &storage.BaselineItem{
			Item: &storage.BaselineItem_ProcessName{
				ProcessName: processName,
			},
		},
		Auto: true,
	}
	return &storage.ProcessBaseline{
		Elements: []*storage.BaselineElement{process},
		Created:  createStamp,
	}
}

// GetScopedProcessBaseline returns a mock ProcessBaseline belonging to the input scope.
func GetScopedProcessBaseline(id string, clusterID string, namespace string) *storage.ProcessBaseline {
	return &storage.ProcessBaseline{
		Id: id,
		Key: &storage.ProcessBaselineKey{
			DeploymentId:  id,
			ClusterId:     clusterID,
			Namespace:     namespace,
			ContainerName: id,
		},
	}
}

// GetProcessBaselineWithID returns an excluded scope with the ID filled out.
func GetProcessBaselineWithID() *storage.ProcessBaseline {
	baseline := GetProcessBaselineWithKey()
	baseline.Id = uuid.NewV4().String()
	return baseline
}

// GetBaselineKey returns a random valid `ProcessBaselineKey`.
func GetBaselineKey() *storage.ProcessBaselineKey {
	return &storage.ProcessBaselineKey{
		DeploymentId:  uuid.NewV4().String(),
		ContainerName: uuid.NewV4().String(),
		ClusterId:     uuid.NewV4().String(),
		Namespace:     uuid.NewV4().String(),
	}
}

// GetProcessBaselineWithKey returns an excluded scope and its key.
func GetProcessBaselineWithKey() *storage.ProcessBaseline {
	key := GetBaselineKey()
	baseline := GetProcessBaseline()
	baseline.Key = key
	return baseline
}

// GetBaselineElement returns a `*storage.BaselineElement` with a given process name.
func GetBaselineElement(processName string) *storage.BaselineElement {
	return &storage.BaselineElement{
		Element: &storage.BaselineItem{
			Item: &storage.BaselineItem_ProcessName{
				ProcessName: processName,
			},
		},
		Auto: true,
	}
}

// MakeBaselineItems turns a list of strings into a
// list of storage objects for more convenient test.
func MakeBaselineItems(strings ...string) []*storage.BaselineItem {
	elements := make([]*storage.BaselineItem, 0, len(strings))
	for _, stringName := range strings {
		elements = append(elements, &storage.BaselineItem{Item: &storage.BaselineItem_ProcessName{ProcessName: stringName}})
	}
	return elements
}

// MakeBaselineElements turns a list of strings into a
// list of storage objects for more convenient test.
func MakeBaselineElements(strings ...string) []*storage.BaselineElement {
	items := MakeBaselineItems(strings...)

	elements := make([]*storage.BaselineElement, 0, len(items))
	for _, item := range items {
		elements = append(elements, &storage.BaselineElement{Element: item})
	}
	return elements
}
