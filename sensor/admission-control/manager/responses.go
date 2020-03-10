package manager

import (
	admission "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func pass(uid types.UID) *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		UID:     uid,
		Allowed: true,
	}
}

func fail(uid types.UID, message string) *admission.AdmissionResponse {
	return &admission.AdmissionResponse{
		UID:     uid,
		Allowed: false,
		Result: &metav1.Status{
			Status:  "Failure",
			Reason:  metav1.StatusReason("Failed currently enforced policies from StackRox"),
			Message: message,
		},
	}
}
