package utils

import (
	"github.com/operator-framework/helm-operator-plugins/pkg/extensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// UpdateStatusCondition returns an UpdateStatusFunc that updates the given condition.
func UpdateStatusCondition(conditionType string, status metav1.ConditionStatus, reason string, message string) extensions.UpdateStatusFunc {
	return func(uSt *unstructured.Unstructured) bool {
		return updateStatusCondition(uSt, conditionType, status, reason, message)
	}
}

func updateStatusCondition(uSt *unstructured.Unstructured, conditionType string, status metav1.ConditionStatus, reason string, message string) bool {
	newCond := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	condsObj, _, err := unstructured.NestedFieldNoCopy(uSt.Object, "conditions")
	if err != nil {
		return false
	}

	conds, ok := condsObj.([]interface{})
	if !ok && condsObj != nil {
		return false // unexpected: conditions found, but is not a slice and not nil
	}

	found := false
	for i, cond := range conds {
		condObj, _ := cond.(map[string]interface{})
		if condObj == nil {
			continue
		}
		var oldCond metav1.Condition
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(condObj, &oldCond); err != nil {
			continue
		}
		if oldCond.Type != newCond.Type {
			continue
		}
		if newCond.Status == oldCond.Status {
			if newCond.Reason == oldCond.Reason && newCond.Message == oldCond.Message {
				return false
			}
			newCond.LastTransitionTime = oldCond.LastTransitionTime
		}
		newCondUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&newCond)
		if err != nil {
			return false
		}
		conds[i] = newCondUnstructured
		found = true
		break
	}

	if !found {
		newCondUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&newCond)
		if err != nil {
			return false
		}
		conds = append(conds, newCondUnstructured)
	}

	if uSt.Object == nil {
		uSt.Object = make(map[string]interface{})
	}

	if err := unstructured.SetNestedSlice(uSt.Object, conds, "conditions"); err != nil {
		return false
	}

	return true
}
