package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

var (
	log = logging.LoggerForModule()
)

//go:generate mockgen-wrapper
type virtualMachineStore interface {
	Get(uid string) *store.VirtualMachineInfo
	AddOrUpdateVirtualMachine(vm *store.VirtualMachineInfo)
	AddOrUpdateVirtualMachineInstance(uid, namespace string, vsockCID *uint32, isRunning bool)
	RemoveVirtualMachine(uid string)
	RemoveVirtualMachineInstance(uid string)
}

type VirtualMachineInstanceDispatcher struct {
	clusterID string
	store     virtualMachineStore
}

func NewVirtualMachineInstanceDispatcher(clusterID string, store virtualMachineStore) *VirtualMachineInstanceDispatcher {
	return &VirtualMachineInstanceDispatcher{
		clusterID: clusterID,
		store:     store,
	}
}

func (d *VirtualMachineInstanceDispatcher) ProcessEvent(
	obj interface{},
	_ interface{},
	action central.ResourceAction,
) *component.ResourceEvent {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		log.Errorf("not of type 'Unstructured': %T", obj)
		return nil
	}
	virtualMachineInstance := &kubeVirtV1.VirtualMachineInstance{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.Object, virtualMachineInstance); err != nil {
		log.Errorf("unable to convert 'Unstructured' to 'VirtualMachineInstance': %v", err)
		return nil
	}
	if virtualMachineInstance.GetUID() == "" {
		log.Errorf("convertion from unstructured failed: %v", obj)
		return nil
	}
	if len(virtualMachineInstance.GetOwnerReferences()) != 1 {
		log.Errorf("virtual machine instance with no owner reference %v", virtualMachineInstance.GetOwnerReferences())
		return nil
	}
	vmUID := string(virtualMachineInstance.GetOwnerReferences()[0].UID)
	ownerEventReceived := d.store.Get(vmUID) != nil
	if action == central.ResourceAction_REMOVE_RESOURCE {
		d.store.RemoveVirtualMachineInstance(vmUID)
	} else {
		isRunning := virtualMachineInstance.Status.Phase == kubeVirtV1.Running
		vsock := virtualMachineInstance.Status.VSOCKCID
		d.store.AddOrUpdateVirtualMachineInstance(vmUID, virtualMachineInstance.Namespace, vsock, isRunning)
	}

	// Do not send UPDATE events if:
	// - we are syncing resources
	// - we did not receive the VirtualMachine associated with this instance yet
	if action == central.ResourceAction_SYNC_RESOURCE || !ownerEventReceived {
		return nil
	}

	vmInfo := d.store.Get(vmUID)
	if vmInfo == nil {
		log.Error("store does not contain the owner reference to this instance")
		return nil
	}

	// VirtualMachineInstances are not tracked by Central.
	// Send an UPDATE event of the VirtualMachine associated with this instance
	return component.NewEvent(&central.SensorEvent{
		Id:     vmInfo.UID,
		Action: central.ResourceAction_UPDATE_RESOURCE,
		Resource: &central.SensorEvent_VirtualMachine{
			VirtualMachine: &virtualMachineV1.VirtualMachine{
				Id:        vmInfo.UID,
				Name:      vmInfo.Name,
				Namespace: vmInfo.Namespace,
				ClusterId: d.clusterID,
			},
		},
	})
}
