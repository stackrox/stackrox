package policyreport

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/policyreport"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/sensor/common/selector"
	"github.com/stackrox/rox/sensor/common/store"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type fakeHierarchy struct {
	parents map[string]set.StringSet
}

func (f *fakeHierarchy) Add(_ metav1.Object)                         {}
func (f *fakeHierarchy) AddManually(_, _ string)                     {}
func (f *fakeHierarchy) Remove(_ string)                             {}
func (f *fakeHierarchy) IsValidChild(_ string, _ metav1.Object) bool { return false }
func (f *fakeHierarchy) TopLevelParents(child string) set.StringSet {
	result := set.NewStringSet()
	f.topLevelRecursive(child, result)
	return result
}

func (f *fakeHierarchy) topLevelRecursive(child string, result set.StringSet) {
	parents, ok := f.parents[child]
	if !ok || parents.Cardinality() == 0 {
		result.Add(child)
		return
	}
	for parent := range parents {
		f.topLevelRecursive(parent, result)
	}
}

type fakeDeploymentStore struct {
	deployments map[string]*storage.Deployment
}

func (f *fakeDeploymentStore) Get(id string) *storage.Deployment {
	return f.deployments[id]
}

func (f *fakeDeploymentStore) GetAll() []*storage.Deployment            { return nil }
func (f *fakeDeploymentStore) GetSnapshot(_ string) *storage.Deployment { return nil }
func (f *fakeDeploymentStore) GetBuiltDeployment(_ string) (*storage.Deployment, bool) {
	return nil, false
}
func (f *fakeDeploymentStore) FindDeploymentIDsWithServiceAccount(_, _ string) []string { return nil }
func (f *fakeDeploymentStore) FindDeploymentIDsByLabels(_ string, _ selector.Selector) []string {
	return nil
}
func (f *fakeDeploymentStore) FindDeploymentIDsByImages(_ []*storage.Image) []string { return nil }
func (f *fakeDeploymentStore) BuildDeploymentWithDependencies(_ string, _ store.Dependencies) (*storage.Deployment, bool, error) {
	return nil, false, nil
}
func (f *fakeDeploymentStore) CountDeploymentsForNamespace(_ string) int { return 0 }
func (f *fakeDeploymentStore) EnhanceDeploymentReadOnly(_ *storage.Deployment, _ store.Dependencies) {
}

func newTestDispatcher(hierarchy *fakeHierarchy, deployStore *fakeDeploymentStore) *Dispatcher {
	if hierarchy == nil {
		hierarchy = &fakeHierarchy{parents: map[string]set.StringSet{}}
	}
	if deployStore == nil {
		deployStore = &fakeDeploymentStore{deployments: map[string]*storage.Deployment{}}
	}
	return NewDispatcher("test-cluster", hierarchy, deployStore)
}

func TestProcessEvent_FeatureDisabled(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "false")
	d := newTestDispatcher(nil, nil)
	result := d.ProcessEvent(&unstructured.Unstructured{}, nil, central.ResourceAction_CREATE_RESOURCE)
	assert.Nil(t, result)
}

func TestProcessEvent_NonUnstructuredObject(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := newTestDispatcher(nil, nil)
	result := d.ProcessEvent("not-an-unstructured", nil, central.ResourceAction_CREATE_RESOURCE)
	assert.Nil(t, result)
}

func TestProcessEvent_RemoveAction(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := newTestDispatcher(nil, nil)

	u := &unstructured.Unstructured{}
	u.SetName("test-report")
	u.SetNamespace("default")

	result := d.ProcessEvent(u, nil, central.ResourceAction_REMOVE_RESOURCE)
	assert.Nil(t, result)
}

func TestProcessEvent_ValidReport(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := newTestDispatcher(nil, nil)

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "wgpolicyk8s.io/v1alpha2",
			"kind":       "PolicyReport",
			"metadata": map[string]interface{}{
				"name":            "test-report",
				"namespace":       "default",
				"uid":             "report-uid-1",
				"resourceVersion": "123",
			},
			"scope": map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "Pod",
				"name":       "my-pod",
				"namespace":  "default",
				"uid":        "pod-uid-1",
			},
			"results": []interface{}{
				map[string]interface{}{
					"policy": "require-labels",
					"rule":   "check-team-label",
					"result": "fail",
					"source": "kyverno",
				},
			},
		},
	}

	// Dry-run: should return nil (no Central forwarding), but not error.
	result := d.ProcessEvent(u, nil, central.ResourceAction_CREATE_RESOURCE)
	assert.Nil(t, result, "dry-run dispatcher should not forward events to Central")
}

func TestProcessEvent_InvalidReport(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := newTestDispatcher(nil, nil)

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "wrong/v1",
			"kind":       "PolicyReport",
			"metadata": map[string]interface{}{
				"name": "bad-report",
			},
		},
	}

	result := d.ProcessEvent(u, nil, central.ResourceAction_CREATE_RESOURCE)
	assert.Nil(t, result)
}

func TestResolveSubject_PodResolved(t *testing.T) {
	hierarchy := &fakeHierarchy{
		parents: map[string]set.StringSet{
			"pod-uid-1": set.NewStringSet("deployment-uid-1"),
		},
	}
	deployStore := &fakeDeploymentStore{
		deployments: map[string]*storage.Deployment{
			"deployment-uid-1": {
				Id:   "deployment-uid-1",
				Name: "my-deployment",
			},
		},
	}
	d := newTestDispatcher(hierarchy, deployStore)

	event := &policyreport.SecurityEvent{
		Subject: policyreport.Subject{
			Kind: "Pod",
			UID:  "pod-uid-1",
			Name: "my-pod",
		},
	}
	d.resolveSubject(event)

	assert.Equal(t, policyreport.EntityTypeDeployment, event.ResolvedEntity.Type)
	assert.Equal(t, "deployment-uid-1", event.ResolvedEntity.ID)
}

func TestResolveSubject_PodUnresolved(t *testing.T) {
	d := newTestDispatcher(nil, nil)

	event := &policyreport.SecurityEvent{
		Subject: policyreport.Subject{
			Kind: "Pod",
			UID:  "unknown-pod",
			Name: "orphan-pod",
		},
	}
	d.resolveSubject(event)

	assert.Equal(t, policyreport.EntityTypeUnknown, event.ResolvedEntity.Type)
	assert.Empty(t, event.ResolvedEntity.ID)
}

func TestResolveSubject_NonPodSubject(t *testing.T) {
	d := newTestDispatcher(nil, nil)

	event := &policyreport.SecurityEvent{
		Subject: policyreport.Subject{
			Kind: "Node",
			UID:  "node-uid-1",
			Name: "my-node",
		},
	}
	d.resolveSubject(event)

	assert.Equal(t, policyreport.EntityTypeUnknown, event.ResolvedEntity.Type)
}

func TestResolveSubject_MultipleOwners(t *testing.T) {
	hierarchy := &fakeHierarchy{
		parents: map[string]set.StringSet{
			"pod-uid-1": set.NewStringSet("deploy-1", "deploy-2"),
		},
	}
	d := newTestDispatcher(hierarchy, nil)

	event := &policyreport.SecurityEvent{
		Subject: policyreport.Subject{
			Kind: "Pod",
			UID:  "pod-uid-1",
			Name: "confused-pod",
		},
	}
	d.resolveSubject(event)

	assert.Equal(t, policyreport.EntityTypeUnknown, event.ResolvedEntity.Type)
}

func TestResolveSubject_OwnerNotInStore(t *testing.T) {
	hierarchy := &fakeHierarchy{
		parents: map[string]set.StringSet{
			"pod-uid-1": set.NewStringSet("missing-deploy"),
		},
	}
	deployStore := &fakeDeploymentStore{
		deployments: map[string]*storage.Deployment{},
	}
	d := newTestDispatcher(hierarchy, deployStore)

	event := &policyreport.SecurityEvent{
		Subject: policyreport.Subject{
			Kind: "Pod",
			UID:  "pod-uid-1",
			Name: "my-pod",
		},
	}
	d.resolveSubject(event)

	assert.Equal(t, policyreport.EntityTypeUnknown, event.ResolvedEntity.Type)
}

func TestResolveSubject_TransitiveOwnership(t *testing.T) {
	hierarchy := &fakeHierarchy{
		parents: map[string]set.StringSet{
			"pod-uid-1": set.NewStringSet("rs-uid-1"),
			"rs-uid-1":  set.NewStringSet("deploy-uid-1"),
		},
	}
	deployStore := &fakeDeploymentStore{
		deployments: map[string]*storage.Deployment{
			"deploy-uid-1": {
				Id:   "deploy-uid-1",
				Name: "my-deployment",
			},
		},
	}
	d := newTestDispatcher(hierarchy, deployStore)

	event := &policyreport.SecurityEvent{
		Subject: policyreport.Subject{
			Kind: "Pod",
			UID:  "pod-uid-1",
			Name: "my-pod",
		},
	}
	d.resolveSubject(event)

	assert.Equal(t, policyreport.EntityTypeDeployment, event.ResolvedEntity.Type)
	assert.Equal(t, "deploy-uid-1", event.ResolvedEntity.ID)
}
