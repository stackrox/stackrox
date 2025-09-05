package dispatcher

import (
	"github.com/stackrox/rox/pkg/virtualmachine"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getVirtualMachineOwnerReference(owners []metav1.OwnerReference) (*metav1.OwnerReference, bool) {
	for _, ref := range owners {
		// There should be only one VirtualMachine OwnerReference
		// VirtualMachines and VirtualMachineInstances map 1:1
		if ref.Kind == virtualmachine.VirtualMachine.Kind {
			return &ref, true
		}
	}
	return nil, false
}
