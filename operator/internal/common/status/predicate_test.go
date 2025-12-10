package status

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
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

	pred := SkipStatusControllerUpdates[*platform.Central]{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pred.Update(event.TypedUpdateEvent[*platform.Central]{
				ObjectOld: tt.old,
				ObjectNew: tt.new,
			})

			if result != tt.shallReconcile {
				t.Errorf("Expected predicate to return %v, got %v", tt.shallReconcile, result)
			}
		})
	}
}
