package policyreport

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestProcessEvent_FeatureDisabled(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "false")
	d := NewDispatcher("test-cluster")
	result := d.ProcessEvent(&unstructured.Unstructured{}, nil, central.ResourceAction_CREATE_RESOURCE)
	assert.Nil(t, result)
}

func TestProcessEvent_NonUnstructuredObject(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := NewDispatcher("test-cluster")
	result := d.ProcessEvent("not-an-unstructured", nil, central.ResourceAction_CREATE_RESOURCE)
	assert.Nil(t, result)
}

func TestProcessEvent_RemoveAction(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := NewDispatcher("test-cluster")

	u := &unstructured.Unstructured{}
	u.SetName("test-report")
	u.SetNamespace("default")

	result := d.ProcessEvent(u, nil, central.ResourceAction_REMOVE_RESOURCE)
	assert.Nil(t, result)
}

func TestProcessEvent_ValidReport(t *testing.T) {
	t.Setenv(features.PolicyReports.EnvVar(), "true")
	d := NewDispatcher("test-cluster")

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
	d := NewDispatcher("test-cluster")

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
