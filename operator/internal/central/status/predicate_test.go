package status

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	platform "github.com/stackrox/rox/operator/api/v1alpha1"
)

func TestCentralStatusPredicate_OwnedConditionsUnchanged_ShouldAllow(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 1,
		},
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
				{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "InstallSuccessful"},
			},
		},
	}

	new := &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 1,
		},
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
				{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "UpgradeSuccessful"}, // Changed
			},
			ObservedGeneration: 5, // Changed
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if !result {
		t.Error("Expected predicate to allow update when owned conditions unchanged but other fields changed")
	}
}

func TestCentralStatusPredicate_AvailableChanged_ShouldBlock(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusFalse, Reason: "DeploymentsNotReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
			},
		},
	}

	new := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"}, // Changed
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
			},
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if result {
		t.Error("Expected predicate to block update when Available condition changed")
	}
}

func TestCentralStatusPredicate_ProgressingChanged_ShouldBlock(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusTrue, Reason: "Reconciling"},
			},
		},
	}

	new := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"}, // Changed
			},
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if result {
		t.Error("Expected predicate to block update when Progressing condition changed")
	}
}

func TestCentralStatusPredicate_BothOwnedChanged_ShouldBlock(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusFalse, Reason: "DeploymentsNotReady"},
				{Type: "Progressing", Status: platform.StatusTrue, Reason: "Reconciling"},
			},
		},
	}

	new := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},       // Changed
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"}, // Changed
			},
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if result {
		t.Error("Expected predicate to block update when both owned conditions changed")
	}
}

func TestCentralStatusPredicate_HelmConditionChanged_ShouldAllow(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
				{Type: platform.ConditionDeployed, Status: platform.StatusUnknown},
			},
		},
	}

	new := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
				{Type: platform.ConditionDeployed, Status: platform.StatusTrue, Reason: "InstallSuccessful"}, // Changed
			},
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if !result {
		t.Error("Expected predicate to allow update when helm condition changed")
	}
}

func TestCentralStatusPredicate_ObservedGenerationChanged_ShouldAllow(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
			},
			ObservedGeneration: 5,
		},
	}

	new := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
			},
			ObservedGeneration: 6, // Changed
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if !result {
		t.Error("Expected predicate to allow update when observedGeneration changed")
	}
}

func TestCentralStatusPredicate_NilObjects_ShouldBlock(t *testing.T) {
	predicate := CentralStatusPredicate{}

	tests := []struct {
		name string
		old  *platform.Central
		new  *platform.Central
	}{
		{
			"old nil",
			nil,
			&platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue},
					},
				},
			},
		},
		{
			"new nil",
			&platform.Central{
				Status: platform.CentralStatus{
					Conditions: []platform.StackRoxCondition{
						{Type: "Available", Status: platform.StatusTrue},
					},
				},
			},
			nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := predicate.Update(event.UpdateEvent{
				ObjectOld: tt.old,
				ObjectNew: tt.new,
			})

			if result {
				t.Errorf("Expected predicate to block update when objects are nil: %s", tt.name)
			}
		})
	}
}

func TestCentralStatusPredicate_BothNil_ShouldBlock(t *testing.T) {
	predicate := CentralStatusPredicate{}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: nil,
		ObjectNew: nil,
	})

	if result {
		t.Error("Expected predicate to block update when both objects are nil")
	}
}

func TestCentralStatusPredicate_SpecChanged_ShouldAllow(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 5,
		},
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
			},
		},
	}

	new := &platform.Central{
		ObjectMeta: metav1.ObjectMeta{
			Generation: 6, // Changed (spec changed)
		},
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: "Available", Status: platform.StatusTrue, Reason: "DeploymentsReady"},
				{Type: "Progressing", Status: platform.StatusFalse, Reason: "ReconcileSuccessful"},
			},
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if !result {
		t.Error("Expected predicate to allow update when spec changed (generation incremented)")
	}
}

func TestCentralStatusPredicate_NoConditions_ShouldAllow(t *testing.T) {
	predicate := CentralStatusPredicate{}

	old := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{},
		},
	}

	new := &platform.Central{
		Status: platform.CentralStatus{
			Conditions: []platform.StackRoxCondition{
				{Type: platform.ConditionDeployed, Status: platform.StatusTrue},
			},
		},
	}

	result := predicate.Update(event.UpdateEvent{
		ObjectOld: old,
		ObjectNew: new,
	})

	if !result {
		t.Error("Expected predicate to allow update when owned conditions don't exist (initial state)")
	}
}
