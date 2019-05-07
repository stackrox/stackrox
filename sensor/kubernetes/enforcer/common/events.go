package common

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

// MarkScaledToZero updates the k8s event logs to reflect that stackrox scaled a deployment to zero.
func MarkScaledToZero(recorder record.EventRecorder, ref *corev1.ObjectReference) error {
	recorder.Event(ref, corev1.EventTypeWarning, "StackRox enforcement", "StackRox scaled the deployment to zero replicas due to a policy violation")
	return nil
}

// MarkNodeConstraintApplied updates the k8s event logs to reflect that stackrox added an unsatisfiable node constraint.
func MarkNodeConstraintApplied(recorder record.EventRecorder, ref *corev1.ObjectReference) error {
	recorder.Event(ref, corev1.EventTypeWarning, "StackRox enforcement", "StackRox applied an unsatisfiable node constraint due to a policy violation")
	return nil
}

// MarkPodKilled updates the k8s event logs to reflect that stackrox deleted a pod.
func MarkPodKilled(recorder record.EventRecorder, ref *corev1.ObjectReference) error {
	recorder.Event(ref, corev1.EventTypeWarning, "StackRox enforcement", "StackRox deleted a pod due to a runtime activity policy violation")
	return nil
}
