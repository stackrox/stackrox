package fixtures

import (
	"time"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/uuid"
)

// Test fixtures for tests involving excluded scopes.

// GetProcessBaseline returns an empty process baseline
// with a random container name and deployment ID.
func GetProcessBaseline() *storage.ProcessBaseline {
	createStamp, _ := protocompat.ConvertTimeToTimestampOrError(time.Now())
	processName := uuid.NewV4().String()
	bi := &storage.BaselineItem{}
	bi.SetProcessName(processName)
	process := &storage.BaselineElement{}
	process.SetElement(bi)
	process.SetAuto(true)
	pb := &storage.ProcessBaseline{}
	pb.SetElements([]*storage.BaselineElement{process})
	pb.SetCreated(createStamp)
	return pb
}

// GetScopedProcessBaseline returns a mock ProcessBaseline belonging to the input scope.
func GetScopedProcessBaseline(id string, clusterID string, namespace string) *storage.ProcessBaseline {
	pbk := &storage.ProcessBaselineKey{}
	pbk.SetDeploymentId(id)
	pbk.SetClusterId(clusterID)
	pbk.SetNamespace(namespace)
	pbk.SetContainerName(id)
	pb := &storage.ProcessBaseline{}
	pb.SetId(id)
	pb.SetKey(pbk)
	return pb
}

// GetProcessBaselineWithID returns an excluded scope with the ID filled out.
func GetProcessBaselineWithID() *storage.ProcessBaseline {
	baseline := GetProcessBaselineWithKey()
	baseline.SetId(uuid.NewV4().String())
	return baseline
}

// GetBaselineKey returns a random valid `ProcessBaselineKey`.
func GetBaselineKey() *storage.ProcessBaselineKey {
	pbk := &storage.ProcessBaselineKey{}
	pbk.SetDeploymentId(uuid.NewV4().String())
	pbk.SetContainerName(uuid.NewV4().String())
	pbk.SetClusterId(uuid.NewV4().String())
	pbk.SetNamespace(uuid.NewV4().String())
	return pbk
}

// GetProcessBaselineWithKey returns an excluded scope and its key.
func GetProcessBaselineWithKey() *storage.ProcessBaseline {
	key := GetBaselineKey()
	baseline := GetProcessBaseline()
	baseline.SetKey(key)
	return baseline
}

// GetBaselineElement returns a `*storage.BaselineElement` with a given process name.
func GetBaselineElement(processName string) *storage.BaselineElement {
	bi := &storage.BaselineItem{}
	bi.SetProcessName(processName)
	be := &storage.BaselineElement{}
	be.SetElement(bi)
	be.SetAuto(true)
	return be
}

// MakeBaselineItems turns a list of strings into a
// list of storage objects for more convenient test.
func MakeBaselineItems(strings ...string) []*storage.BaselineItem {
	elements := make([]*storage.BaselineItem, 0, len(strings))
	for _, stringName := range strings {
		bi := &storage.BaselineItem{}
		bi.SetProcessName(stringName)
		elements = append(elements, bi)
	}
	return elements
}

// MakeBaselineElements turns a list of strings into a
// list of storage objects for more convenient test.
func MakeBaselineElements(strings ...string) []*storage.BaselineElement {
	items := MakeBaselineItems(strings...)

	elements := make([]*storage.BaselineElement, 0, len(items))
	for _, item := range items {
		be := &storage.BaselineElement{}
		be.SetElement(item)
		elements = append(elements, be)
	}
	return elements
}
