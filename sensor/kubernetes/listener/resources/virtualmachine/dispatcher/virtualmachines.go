package dispatcher

import (
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/utils"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

const (
	GuestOSKey     = "Guest OS"
	DescriptionKey = "Description"
	IPAddressesKey = "IP Addresses"
	ActivePodsKey  = "Active Pods"
	NodeNameKey    = "Node Name"
	BootOrderKey   = "Boot Order"
	CDRomDisksKey  = "CD-ROM Disks"
	UnknownGuestOS = "unknown"
)

type VirtualMachineDispatcher struct {
	clusterID string
	store     virtualMachineStore
}

func NewVirtualMachineDispatcher(clusterID string, store virtualMachineStore) *VirtualMachineDispatcher {
	return &VirtualMachineDispatcher{
		clusterID: clusterID,
		store:     store,
	}
}

func (d *VirtualMachineDispatcher) ProcessEvent(
	obj interface{},
	_ interface{},
	action central.ResourceAction,
) *component.ResourceEvent {
	if !features.VirtualMachines.Enabled() || !centralcaps.Has(centralsensor.VirtualMachinesSupported) {
		return nil
	}
	virtualMachine := &kubeVirtV1.VirtualMachine{}
	if err := utils.FromUnstructuredToSpecificTypePointer(obj, virtualMachine); err != nil {
		log.Errorf("unable to convert 'Unstructured' to 'VirtualMachine': %v", err)
		return nil
	}
	if virtualMachine.GetUID() == "" {
		log.Errorf("conversion from 'Unstructured' to '%T' failed: %v", virtualMachine, obj)
		return nil
	}
	isRunning := virtualMachine.Status.PrintableStatus == kubeVirtV1.VirtualMachineStatusRunning
	disks := extractDisksFromVM(virtualMachine)
	vm := &virtualmachine.Info{
		ID:          virtualmachine.VMID(virtualMachine.GetUID()),
		Name:        virtualMachine.GetName(),
		Namespace:   virtualMachine.GetNamespace(),
		Running:     isRunning,
		Description: descriptionFromAnnotations(virtualMachine.GetAnnotations()),
		BootOrder:   extractBootOrder(disks),
		CDRomDisks:  extractCDRomDisks(disks),
	}
	return processVirtualMachine(vm, action, d.clusterID, d.store)
}

func processVirtualMachine(vm *virtualmachine.Info, action central.ResourceAction, clusterID string, store virtualMachineStore) *component.ResourceEvent {
	if action == central.ResourceAction_REMOVE_RESOURCE {
		store.Remove(vm.ID)
	} else {
		vm = store.AddOrUpdate(vm)
	}
	return component.NewEvent(createEvent(action, clusterID, vm))
}
