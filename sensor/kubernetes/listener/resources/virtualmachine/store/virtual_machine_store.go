package store

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

type VMID string

// VirtualMachineInfo information about a VirtualMachine
type VirtualMachineInfo struct {
	ID        VMID
	Name      string
	Namespace string
	VSOCKCID  *uint32
	Running   bool
}

// Copy returns a copy of the VirtualMachineInfo
func (v *VirtualMachineInfo) Copy() *VirtualMachineInfo {
	if v == nil {
		return nil
	}
	ret := &VirtualMachineInfo{
		ID:        v.ID,
		Name:      v.Name,
		Namespace: v.Namespace,
		Running:   v.Running,
	}
	if v.VSOCKCID != nil {
		vsockCIDValue := *v.VSOCKCID
		ret.VSOCKCID = &vsockCIDValue
	}
	return ret
}

// VirtualMachineStore stores the information about the VirtualMachines in the cluster
type VirtualMachineStore struct {
	lock sync.RWMutex

	cidToID         map[uint32]VMID
	idToCID         map[VMID]uint32
	namespaceToID   map[string]set.Set[VMID]
	virtualMachines map[VMID]*VirtualMachineInfo
}

// NewVirtualMachineStore returns a new store
func NewVirtualMachineStore() *VirtualMachineStore {
	return &VirtualMachineStore{
		virtualMachines: make(map[VMID]*VirtualMachineInfo),
		namespaceToID:   make(map[string]set.Set[VMID]),
		cidToID:         make(map[uint32]VMID),
		idToCID:         make(map[VMID]uint32),
	}
}

// AddOrUpdate upserts a new VirtualMachine
func (s *VirtualMachineStore) AddOrUpdate(vm *VirtualMachineInfo) {
	if vm == nil {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addOrUpdateNoLock(vm)
}

// UpdateStateOrCreate updates the VirtualMachine state
// If the VirtualMachine is not present we create a new VirtualMachine
func (s *VirtualMachineStore) UpdateStateOrCreate(vm *VirtualMachineInfo) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.updateStatusOrCreateNoLock(vm)
}

// Remove removes a VirtualMachine
func (s *VirtualMachineStore) Remove(id VMID) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeNoLock(id)
}

// ClearState removes a VirtualMachineInstance
func (s *VirtualMachineStore) ClearState(id VMID) {
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

// OnNamespaceDeleted removes the VirtualMachines in the given namespace
func (s *VirtualMachineStore) OnNamespaceDeleted(namespace string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	vmIDsByNamespace := s.namespaceToID[namespace]
	for vmID := range vmIDsByNamespace {
		s.removeNoLock(vmID)
	}
}

// Get returns the VirtualMachineInfo associated with the given ID
func (s *VirtualMachineStore) Get(id VMID) *VirtualMachineInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.virtualMachines[id].Copy()
}

// Has returns true if the store contains the VirtualMachine with the given ID
func (s *VirtualMachineStore) Has(id VMID) bool {
	return s.Get(id) != nil
}

func (s *VirtualMachineStore) addOrUpdateNoLock(vm *VirtualMachineInfo) {
	// Replace VSOCK info
	// If the new VirtualMachineInfo (vm) does not have a VSOCK,
	// then we use the previous value
	vm.VSOCKCID = s.replaceVSOCKInfoNoLock(vm)

	// Upsert the VirtualMachineInfo
	vmIDsByNamespace := s.getOrCreateNamespaceSet(vm.Namespace)
	vmIDsByNamespace.Add(vm.ID)
	s.virtualMachines[vm.ID] = vm
}

func (s *VirtualMachineStore) getOrCreateNamespaceSet(namespace string) set.Set[VMID] {
	vmIDsByNamespace, found := s.namespaceToID[namespace]
	if !found {
		vmIDsByNamespace = set.NewSet[VMID]()
		s.namespaceToID[namespace] = vmIDsByNamespace
	}
	return vmIDsByNamespace
}

func (s *VirtualMachineStore) updateStatusOrCreateNoLock(updateInfo *VirtualMachineInfo) {
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

func (s *VirtualMachineStore) addOrUpdateVSOCKInfoNoLock(id VMID, vsockCID *uint32) *uint32 {
	if vsockCID == nil {
		return nil
	}
	s.idToCID[id] = *vsockCID
	s.cidToID[*vsockCID] = id
	// copy value before return
	val := *vsockCID
	return &val
}

func (s *VirtualMachineStore) removeVSOCKInfoNoLock(id VMID, vsockCID *uint32) {
	if vsockCID == nil {
		return
	}
	delete(s.idToCID, id)
	delete(s.cidToID, *vsockCID)
}

func (s *VirtualMachineStore) replaceVSOCKInfoNoLock(vm *VirtualMachineInfo) *uint32 {
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

func (s *VirtualMachineStore) removeNoLock(id VMID) {
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

func (s *VirtualMachineStore) clearStatusNoLock(id VMID) {
	vm, ok := s.virtualMachines[id]
	if !ok {
		return
	}
	s.removeVSOCKInfoNoLock(vm.ID, vm.VSOCKCID)
	vm.VSOCKCID = nil
	// If the instance is removed the VirtualMachine will transition to Stopped
	vm.Running = false
}
