package store

import (
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

	cidToID         map[uint32]virtualmachine.VMID
	idToCID         map[virtualmachine.VMID]uint32
	namespaceToID   map[string]set.Set[virtualmachine.VMID]
	virtualMachines map[virtualmachine.VMID]*virtualmachine.Info
}

// NewVirtualMachineStore returns a new store
func NewVirtualMachineStore() *VirtualMachineStore {
	return &VirtualMachineStore{
		virtualMachines: make(map[virtualmachine.VMID]*virtualmachine.Info),
		namespaceToID:   make(map[string]set.Set[virtualmachine.VMID]),
		cidToID:         make(map[uint32]virtualmachine.VMID),
		idToCID:         make(map[virtualmachine.VMID]uint32),
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
	clear(s.cidToID)
	clear(s.idToCID)
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

// GetFromCID returns the VirtualMachineInfo associated with a given VSOCK CID
func (s *VirtualMachineStore) GetFromCID(cid uint32) *virtualmachine.Info {
	s.lock.RLock()
	defer s.lock.RUnlock()
	uid, ok := s.cidToID[cid]
	if !ok {
		return nil
	}
	return s.virtualMachines[uid].Copy()
}

func (s *VirtualMachineStore) addOrUpdateNoLock(vm *virtualmachine.Info) {
	// Replace VSOCK info
	// If the new VirtualMachineInfo (vm) does not have a VSOCK,
	// then we use the previous value
	vm.VSOCKCID = s.replaceVSOCKInfoNoLock(vm)

	// Upsert the VirtualMachineInfo
	vmIDsByNamespace := s.getOrCreateNamespaceSet(vm.Namespace)
	vmIDsByNamespace.Add(vm.ID)
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
	// Remove previous VSOCK info
	s.removeVSOCKInfoNoLock(prev.ID, prev.VSOCKCID)
	// Update new VSOCK maps
	prev.VSOCKCID = s.addOrUpdateVSOCKInfoNoLock(updateInfo.ID, updateInfo.VSOCKCID)
	prev.Running = updateInfo.Running
}

func (s *VirtualMachineStore) addOrUpdateVSOCKInfoNoLock(id virtualmachine.VMID, vsockCID *uint32) *uint32 {
	if vsockCID == nil {
		return nil
	}
	s.idToCID[id] = *vsockCID
	s.cidToID[*vsockCID] = id
	// copy value before return
	val := *vsockCID
	return &val
}

func (s *VirtualMachineStore) removeVSOCKInfoNoLock(id virtualmachine.VMID, vsockCID *uint32) {
	if vsockCID == nil {
		return
	}
	delete(s.idToCID, id)
	delete(s.cidToID, *vsockCID)
}

func (s *VirtualMachineStore) replaceVSOCKInfoNoLock(vm *virtualmachine.Info) *uint32 {
	// Remove previous VSOCK info
	// This is needed in case the VSOCK value updates
	prev, found := s.virtualMachines[vm.ID]
	if found {
		s.removeVSOCKInfoNoLock(vm.ID, prev.VSOCKCID)
	}
	// Update VSOCKCID info
	if vm.VSOCKCID == nil && prev != nil {
		vm.VSOCKCID = prev.VSOCKCID
	}
	// Upsert VSOCKCID info
	if vm.VSOCKCID != nil {
		_ = s.addOrUpdateVSOCKInfoNoLock(vm.ID, vm.VSOCKCID)
	}
	return vm.VSOCKCID
}

func (s *VirtualMachineStore) removeNoLock(id virtualmachine.VMID) {
	vm, found := s.virtualMachines[id]
	if !found {
		return
	}
	delete(s.virtualMachines, vm.ID)
	s.removeVSOCKInfoNoLock(vm.ID, vm.VSOCKCID)
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
	s.removeVSOCKInfoNoLock(vm.ID, vm.VSOCKCID)
	vm.VSOCKCID = nil
	// If the instance is removed the VirtualMachine will transition to Stopped
	vm.Running = false
}
