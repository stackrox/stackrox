package store

import (
	"maps"

	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/sensor/common/virtualmachine"
)

var (
	log = logging.LoggerForModule()
)

// VirtualMachineStore stores the information about the VirtualMachines in the cluster
type VirtualMachineStore struct {
	lock sync.RWMutex

	namespaceToID   map[string]set.Set[virtualmachine.VMID]
	virtualMachines map[virtualmachine.VMID]*virtualmachine.Info
}

// NewVirtualMachineStore returns a new store
func NewVirtualMachineStore() *VirtualMachineStore {
	return &VirtualMachineStore{
		virtualMachines: make(map[virtualmachine.VMID]*virtualmachine.Info),
		namespaceToID:   make(map[string]set.Set[virtualmachine.VMID]),
	}
}

// AddOrUpdate upserts a new VirtualMachine
func (s *VirtualMachineStore) AddOrUpdate(vm *virtualmachine.Info) *virtualmachine.Info {
	if vm == nil {
		return nil
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	oldVM := s.virtualMachines[vm.ID]
	if oldVM != nil {
		vm.Running = oldVM.Running
		if oldVM.VSOCKCID != nil {
			vSockCID := *oldVM.VSOCKCID
			vm.VSOCKCID = &vSockCID
		}
		vm.GuestOS = oldVM.GuestOS
		vm.IPAddresses = copyStringSlice(oldVM.IPAddresses)
		vm.ActivePods = copyStringSlice(oldVM.ActivePods)
		vm.NodeName = oldVM.NodeName
		if vm.AgentFacts == nil {
			vm.AgentFacts = oldVM.AgentFacts
		}
	}
	s.addOrUpdateNoLock(vm)
	return vm
}

// UpdateStateOrCreate updates the VirtualMachine state
// If the VirtualMachine is not present we create a new VirtualMachine
func (s *VirtualMachineStore) UpdateStateOrCreate(vm *virtualmachine.Info) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.updateStatusOrCreateNoLock(vm)
}

// Remove removes a VirtualMachine
func (s *VirtualMachineStore) Remove(id virtualmachine.VMID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeNoLock(id)
}

// ClearState removes a VirtualMachineInstance
func (s *VirtualMachineStore) ClearState(id virtualmachine.VMID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clearStatusNoLock(id)
}

// Cleanup resets the store
func (s *VirtualMachineStore) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	clear(s.virtualMachines)
	clear(s.namespaceToID)
}

// OnNamespaceDeleted removes the VirtualMachines in the given namespace.
// This is called when the namespace is getting deleted.
// By that point Sensor should have received all the REMOVE events for the VMs.
// This is here to not leak any resources in case a REMOVE event is lost.
func (s *VirtualMachineStore) OnNamespaceDeleted(namespace string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	vmIDsByNamespace := s.namespaceToID[namespace]
	for vmID := range vmIDsByNamespace {
		s.removeNoLock(vmID)
	}
}

// Get returns the VirtualMachineInfo associated with the given ID
func (s *VirtualMachineStore) Get(id virtualmachine.VMID) *virtualmachine.Info {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.virtualMachines[id].Copy()
}

// Has returns true if the store contains the VirtualMachine with the given ID
func (s *VirtualMachineStore) Has(id virtualmachine.VMID) bool {
	return s.Get(id) != nil
}

// Size returns the number of VirtualMachines in the store
func (s *VirtualMachineStore) Size() int {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return len(s.virtualMachines)
}

// ListRunning returns copies of all VMs currently in running state.
func (s *VirtualMachineStore) ListRunning() []*virtualmachine.Info {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var out []*virtualmachine.Info
	for _, vm := range s.virtualMachines {
		if vm.Running {
			out = append(out, vm.Copy())
		}
	}
	return out
}

func (s *VirtualMachineStore) addOrUpdateNoLock(vm *virtualmachine.Info) {
	vm.VSOCKCID = s.replaceVSOCKInfoNoLock(vm)

	// Upsert the VirtualMachineInfo
	vmIDsByNamespace := s.getOrCreateNamespaceSet(vm.Namespace)
	vmIDsByNamespace.Add(vm.ID)
	vm.AgentFacts = maps.Clone(vm.AgentFacts)
	s.virtualMachines[vm.ID] = vm
}

func (s *VirtualMachineStore) getOrCreateNamespaceSet(namespace string) set.Set[virtualmachine.VMID] {
	vmIDsByNamespace, found := s.namespaceToID[namespace]
	if !found {
		vmIDsByNamespace = set.NewSet[virtualmachine.VMID]()
		s.namespaceToID[namespace] = vmIDsByNamespace
	}
	return vmIDsByNamespace
}

func (s *VirtualMachineStore) updateStatusOrCreateNoLock(updateInfo *virtualmachine.Info) {
	prev, found := s.virtualMachines[updateInfo.ID]
	// This is needed in case of a race between the dispatchers
	if !found {
		// If there is no match, treat this as a normal upsert
		s.addOrUpdateNoLock(updateInfo)
		return
	}
	prev.VSOCKCID = copyVSOCKCID(updateInfo.VSOCKCID)
	prev.Running = updateInfo.Running
	prev.GuestOS = updateInfo.GuestOS
	prev.IPAddresses = copyStringSlice(updateInfo.IPAddresses)
	prev.ActivePods = copyStringSlice(updateInfo.ActivePods)
	prev.NodeName = updateInfo.NodeName
	prev.Description = updateInfo.Description
	prev.BootOrder = copyStringSlice(updateInfo.BootOrder)
	prev.CDRomDisks = copyStringSlice(updateInfo.CDRomDisks)
	if updateInfo.AgentFacts != nil {
		prev.AgentFacts = maps.Clone(updateInfo.AgentFacts)
	}
}

// copyVSOCKCID returns a new pointer so later changes to the caller's value
// cannot change what the store holds.
func copyVSOCKCID(cid *uint32) *uint32 {
	if cid == nil {
		return nil
	}
	return new(*cid)
}

func (s *VirtualMachineStore) replaceVSOCKInfoNoLock(vm *virtualmachine.Info) *uint32 {
	prev := s.virtualMachines[vm.ID]
	// A VM update may omit CID; keep the last known value.
	if vm.VSOCKCID == nil && prev != nil {
		vm.VSOCKCID = prev.VSOCKCID
	}
	return copyVSOCKCID(vm.VSOCKCID)
}

func (s *VirtualMachineStore) removeNoLock(id virtualmachine.VMID) {
	vm, found := s.virtualMachines[id]
	if !found {
		return
	}
	delete(s.virtualMachines, vm.ID)
	vmIDsByNamespace, found := s.namespaceToID[vm.Namespace]
	if !found {
		log.Errorf("namespace %q was not found", vm.Namespace)
		return
	}
	vmIDsByNamespace.Remove(vm.ID)
	if len(vmIDsByNamespace) == 0 {
		delete(s.namespaceToID, vm.Namespace)
	}
}

func (s *VirtualMachineStore) clearStatusNoLock(id virtualmachine.VMID) {
	vm, ok := s.virtualMachines[id]
	if !ok {
		return
	}
	vm.VSOCKCID = nil
	// If the instance is removed the VirtualMachine will transition to Stopped
	vm.Running = false
	vm.GuestOS = ""
	vm.NodeName = ""
	vm.IPAddresses = nil
	vm.ActivePods = nil
}

func copyStringSlice(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	return append([]string(nil), values...)
}
