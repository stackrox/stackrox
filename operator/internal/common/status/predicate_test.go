package status

import (
	"testing"

	"github.com/go-logr/logr"
	platform "github.com/stackrox/rox/operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/utils/ptr"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestCentralStatusControllerUpdatePredicate(t *testing.T) {
	tests := []struct {
		name           string
		old            *platform.Central
		new            *platform.Central
		shallReconcile bool
	}{
		{
			name: "owned conditions unchanged should allow reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
						{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "InstallSuccessful"},
					},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
						{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "UpgradeSuccessful"}, // Changed
					},
				},
			},
			shallReconcile: true,
		},
		{
			name: "Available changed should skip reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusFalse, Reason: "DeploymentsNotReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
					},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"}, // Changed
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
					},
				},
			},
			shallReconcile: false,
		},
		{
			name: "Processing changed should skip reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusTrue, Reason: "Reconciling"},
					},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"}, // Changed
					},
				},
			},
			shallReconcile: false,
		},
		{
			name: "Available and Progressing changed should skip reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusFalse, Reason: "DeploymentsNotReady"},
						{Type: "Progressing", Status: platform.StatusTrue, Reason: "Reconciling"},
					},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},       // Changed
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"}, // Changed
					},
				},
			},
			shallReconcile: false,
		},
		{
			name: "Helm condition changed should allow reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
						{Type: platform.ConditionDeployed, Status: platform.StatusUnknown},
					},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
						{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "InstallSuccessful"}, // Changed
					},
				},
			},
			shallReconcile: true,
		},
		{
			name: "Helm condition changed combined with Available condition changed should allow reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: platform.ConditionDeployed, Status: platform.StatusUnknown},
					},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusFalse, Reason: "DeploymentsNotReady"},             // Changed
						{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "InstallSuccessful"}, // Changed
					},
				},
			},
			shallReconcile: true,
		},
		{
			name: "observedGeneration changed should allow reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
					},
					ObservedGeneration: 5,
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
					},
					ObservedGeneration: 6, // Changed
				},
			},
			shallReconcile: true,
		},
		{
			name: "old object nil should allow reconciliation",
			old:  nil,
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue},
					},
				},
			},
			shallReconcile: true,
		},
		{
			name: "new object nil should allow reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue},
					},
				},
			},
			new:            nil,
			shallReconcile: true,
		},
		{
			name:           "both objects nil should allow reconciliation",
			old:            nil,
			new:            nil,
			shallReconcile: true,
		},
		{
			name: "spec change should allow reconciliation",
			old: &platform.Central{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 5,
				},
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
					},
				},
			},
			new: &platform.Central{
				ObjectMeta: metav1.ObjectMeta{
					Generation: 6, // Changed (spec changed)
				},
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
						{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
					},
				},
			},
			shallReconcile: true,
		},

		{
			name: "no status controller owned conditions should allow reconciliation",
			old: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{},
				},
			},
			new: &platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: platform.ConditionDeployed, Status: platform.StatusTrue},
					},
				},
			},
			shallReconcile: true,
		},
	}

	pred := NewSkipStatusControllerUpdates(logr.Discard(), "Central")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pred.Update(event.UpdateEvent{
				ObjectOld: toUnstructuredCentral(t, tt.old),
				ObjectNew: toUnstructuredCentral(t, tt.new),
			})

			if result != tt.shallReconcile {
				t.Errorf("Expected predicate to return %v, got %v", tt.shallReconcile, result)
			}
		})
	}
}

// toUnstructuredCentral converts a typed Central object to an unstructured object.
// Returns nil if the input is nil.
func toUnstructuredCentral(t *testing.T, central *platform.Central) ctrlClient.Object {
	if central == nil {
		return nil
	}

	objMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(central)
	if err != nil {
		t.Fatalf("Failed to convert Central to unstructured: %v", err)
	}

	u := &unstructured.Unstructured{Object: objMap}
	u.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "platform.stackrox.io",
		Version: "v1alpha1",
		Kind:    "Central",
	})

	return u
}

func TestDeploymentStatusUpdatePredicate(t *testing.T) {
	tests := []struct {
		name           string
		old            *appsv1.Deployment
		new            *appsv1.Deployment
		shallReconcile bool
	}{
		{
			name: "spec.replicas changed should NOT trigger reconciliation",
			old: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(3)),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			new: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(5)), // Changed by HPA
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			shallReconcile: false,
		},
		{
			name: "status.replicas changed should trigger reconciliation",
			old: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			new: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:          5,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			shallReconcile: true,
		},
		{
			name: "status.readyReplicas changed should trigger reconciliation",
			old: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
			},
			new: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 2,
				},
			},
			shallReconcile: true,
		},
		{
			name: "deployment condition changed should trigger reconciliation",
			old: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas: 3,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionFalse},
					},
				},
			},
			new: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas: 3,
					Conditions: []appsv1.DeploymentCondition{
						{Type: appsv1.DeploymentAvailable, Status: corev1.ConditionTrue},
					},
				},
			},
			shallReconcile: true,
		},
		{
			name: "no changes should NOT trigger reconciliation",
			old: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			new: &appsv1.Deployment{
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			shallReconcile: false,
		},
		{
			name: "spec and status both changed should trigger reconciliation",
			old: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(3)),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          3,
					ReadyReplicas:     3,
					AvailableReplicas: 3,
				},
			},
			new: &appsv1.Deployment{
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(5)),
				},
				Status: appsv1.DeploymentStatus{
					Replicas:          5,
					ReadyReplicas:     5,
					AvailableReplicas: 5,
				},
			},
			shallReconcile: true,
		},
		{
			name:           "old object nil should allow reconciliation",
			old:            nil,
			new:            &appsv1.Deployment{},
			shallReconcile: true,
		},
		{
			name:           "new object nil should allow reconciliation",
			old:            &appsv1.Deployment{},
			new:            nil,
			shallReconcile: true,
		},
		{
			name:           "both objects nil should allow reconciliation",
			old:            nil,
			new:            nil,
			shallReconcile: true,
		},
	}

	pred := PassThroughUpdatedStatusPredicate{logger: logr.Discard()}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pred.Update(event.TypedUpdateEvent[*appsv1.Deployment]{
				ObjectOld: tt.old,
				ObjectNew: tt.new,
			})

			if result != tt.shallReconcile {
				t.Errorf("Expected predicate to return %v, got %v", tt.shallReconcile, result)
			}
		})
	}
}

// TestUnstructuredStatusControllerUpdatePredicate verifies that the predicate correctly handles
// unstructured objects (as sent by the helm reconciler) by converting them to typed objects
func TestUnstructuredStatusControllerUpdatePredicate(t *testing.T) {
	tests := []struct {
		name string
		// If testWithFixedKind is false, the test case will be executed for both kinds (Central and SecuredCluster)
		// by setting the kind in the unstructured objects accordingly.
		// If it is true, the test case is expected to be kind-specific.
		testWithFixedKind bool
		old               *unstructured.Unstructured
		new               *unstructured.Unstructured
		shallReconcile    bool
	}{
		{
			name: "Available condition changed should skip reconciliation",
			old: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"metadata": map[string]interface{}{
						"name":      "stackrox",
						"namespace": "stackrox",
					},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Available",
								"status": "False",
								"reason": "DeploymentsNotReady",
							},
						},
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"metadata": map[string]interface{}{
						"name":      "stackrox",
						"namespace": "stackrox",
					},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Available",
								"status": "True",
								"reason": "DeploymentsReady",
							},
						},
					},
				},
			},
			shallReconcile: false,
		},
		{
			name: "Deployed condition changed should allow reconciliation",
			old: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"metadata": map[string]interface{}{
						"name":      "stackrox",
						"namespace": "stackrox",
					},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Available",
								"status": "True",
								"reason": "DeploymentsReady",
							},
							map[string]interface{}{
								"type":   "Deployed",
								"status": "True",
								"reason": "InstallSuccessful",
							},
						},
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"metadata": map[string]interface{}{
						"name":      "stackrox",
						"namespace": "stackrox",
					},
					"status": map[string]interface{}{
						"conditions": []interface{}{
							map[string]interface{}{
								"type":   "Available",
								"status": "True",
								"reason": "DeploymentsReady",
							},
							map[string]interface{}{
								"type":   "Deployed",
								"status": "True",
								"reason": "UpgradeSuccessful",
							},
						},
					},
				},
			},
			shallReconcile: true,
		},
		{
			name:              "Central spec change should allow reconciliation",
			testWithFixedKind: true,
			old: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"kind":       "Central",
					"metadata": map[string]interface{}{
						"name":      "stackrox-central-services",
						"namespace": "acs-central",
					},
					"spec": map[string]interface{}{
						"central": map[string]interface{}{
							"persistence": map[string]interface{}{
								"persistentVolumeClaim": map[string]interface{}{
									"size": "100Gi",
								},
							},
						},
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"kind":       "Central",
					"metadata": map[string]interface{}{
						"name":      "stackrox-central-services",
						"namespace": "acs-central",
					},
					"spec": map[string]interface{}{
						"central": map[string]interface{}{
							"persistence": map[string]interface{}{
								"persistentVolumeClaim": map[string]interface{}{
									"size": "200Gi", // Changed
								},
							},
						},
					},
				},
			},
			shallReconcile: true,
		},
		{
			name:              "SecuredCluster spec change should allow reconciliation",
			testWithFixedKind: true,
			old: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"kind":       "SecuredCluster",
					"metadata": map[string]interface{}{
						"name":      "stackrox-secured-cluster-services",
						"namespace": "stackrox",
					},
					"spec": map[string]interface{}{
						"clusterName": "cluster1",
					},
				},
			},
			new: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "platform.stackrox.io/v1alpha1",
					"kind":       "SecuredCluster",
					"metadata": map[string]interface{}{
						"name":      "stackrox-secured-cluster-services",
						"namespace": "stackrox",
					},
					"spec": map[string]interface{}{
						"clusterName": "cluster2",
					},
				},
			},
			shallReconcile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runWithFixedKind := func(old, new *unstructured.Unstructured, shallReconcile bool) {
				pred := NewSkipStatusControllerUpdates(logr.Discard(), old.GetKind())
				result := pred.Update(event.UpdateEvent{
					ObjectOld: old,
					ObjectNew: new,
				})
				if result != shallReconcile {
					t.Errorf("Expected predicate to return %v, got %v", tt.shallReconcile, result)
				}
			}

			// We use this loop for "instantiating" all the test cases with the correct kind
			// without duplicating the test cases themselves for Central & SecuredCluster:
			//
			// The kind == "" case corresponds to the test cases which have a fixed kind set in
			// the unstructured object already.
			//
			// The kind == "Central" and kind == "SecuredCluster" are only executed for the
			// test cases which do not have a fixed kind set -- in those cases we set the kind
			// in the unstructured objects correspondingly.
			for _, kind := range []string{"", "Central", "SecuredCluster"} {
				old := tt.old.DeepCopy()
				new := tt.new.DeepCopy()

				if kind == "" {
					if !tt.testWithFixedKind {
						continue
					}
				} else {
					if tt.testWithFixedKind {
						continue
					}
					old.SetKind(kind)
					new.SetKind(kind)
				}

				runWithFixedKind(old, new, tt.shallReconcile)
			}
		})
	}
}
