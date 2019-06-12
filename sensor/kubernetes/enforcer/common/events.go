package common

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

// MarkScaledToZero updates the k8s event logs to reflect that stackrox scaled a deployment to zero.
func MarkScaledToZero(recorder record.EventRecorder, policyName string, ref *corev1.ObjectReference) error {
	message := fmt.Sprintf("Deployment violated StackRox policy %q and was scaled down", policyName)
	recorder.Event(ref, corev1.EventTypeWarning, "StackRox enforcement", message)
	return nil
}

// MarkNodeConstraintApplied updates the k8s event logs to reflect that stackrox added an unsatisfiable node constraint.
func MarkNodeConstraintApplied(recorder record.EventRecorder, policyName string, ref *corev1.ObjectReference) error {
	message := fmt.Sprintf("Deployment violated StackRox policy %q and had an unsatisfiable node constraint added", policyName)
	recorder.Event(ref, corev1.EventTypeWarning, "StackRox enforcement", message)
	return nil
}

// MarkPodKilled updates the k8s event logs to reflect that stackrox deleted a pod.
func MarkPodKilled(recorder record.EventRecorder, policyName string, ref *corev1.ObjectReference) error {
	message := fmt.Sprintf("Pod violated StackRox policy %q and was killed", policyName)
	recorder.Event(ref, corev1.EventTypeWarning, "StackRox enforcement", message)
	return nil
}
