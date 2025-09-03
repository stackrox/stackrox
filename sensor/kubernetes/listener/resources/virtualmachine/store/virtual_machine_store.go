package store

import (
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stackrox/rox/pkg/sync"
)

var (
	log = logging.LoggerForModule()
)

// VirtualMachineInfo information about a VirtualMachine
type VirtualMachineInfo struct {
	UID       string
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
		UID:       v.UID,
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

	cidToUID        map[uint32]string
	uidToCID        map[string]uint32
	namespaceToUID  map[string]set.StringSet
	virtualMachines map[string]*VirtualMachineInfo
}

// NewVirtualMachineStore returns a new store
func NewVirtualMachineStore() *VirtualMachineStore {
	return &VirtualMachineStore{
		virtualMachines: make(map[string]*VirtualMachineInfo),
		namespaceToUID:  make(map[string]set.StringSet),
		cidToUID:        make(map[uint32]string),
		uidToCID:        make(map[string]uint32),
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
func (s *VirtualMachineStore) Remove(uid string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeNoLock(uid)
}

// ClearState removes a VirtualMachineInstance
func (s *VirtualMachineStore) ClearState(uid string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clearStatusNoLock(uid)
}

// Cleanup resets the store
func (s *VirtualMachineStore) Cleanup() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.virtualMachines = make(map[string]*VirtualMachineInfo)
	s.namespaceToUID = make(map[string]set.StringSet)
	s.cidToUID = make(map[uint32]string)
	s.uidToCID = make(map[string]uint32)
}

// OnNamespaceDeleted removes the VirtualMachines in the given namespace
func (s *VirtualMachineStore) OnNamespaceDeleted(namespace string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	vmIDsByNamespace := s.namespaceToUID[namespace]
	for vmID := range vmIDsByNamespace {
		s.removeNoLock(vmID)
	}
}

// Get returns the VirtualMachineInfo associated with the given UID
func (s *VirtualMachineStore) Get(uid string) *VirtualMachineInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.virtualMachines[uid].Copy()
}

// Has returns true if the store contains the VirtualMachine with the given UID
func (s *VirtualMachineStore) Has(uid string) bool {
	return s.Get(uid) != nil
}

func (s *VirtualMachineStore) addOrUpdateNoLock(vm *VirtualMachineInfo) {
	log.Debugf("Pushing virtual machine to store: name %q UID: %q", vm.Name, vm.UID)
	// Remove previous VSOCK info
	// This is needed in case of races between the dispatchers
	prev, found := s.virtualMachines[vm.UID]
	if found {
		s.removeVSOCKInfoNoLock(vm.UID, prev.VSOCKCID)
	}

	// Update VSOCK info
	if vm.VSOCKCID == nil && prev != nil {
		vm.VSOCKCID = prev.VSOCKCID
	}
	// Upsert VSOCK info
	if vm.VSOCKCID != nil {
		_ = s.addOrUpdateVSOCKInfoNoLock(vm.UID, vm.VSOCKCID)
	}

	// Upsert the VirtualMachineInfo
	vmIDsByNamespace := s.getOrCreateNamespaceSet(vm.Namespace)
	vmIDsByNamespace.Add(vm.UID)
	s.virtualMachines[vm.UID] = vm
}

func (s *VirtualMachineStore) getOrCreateNamespaceSet(namespace string) set.StringSet {
	vmIDsByNamespace, found := s.namespaceToUID[namespace]
	if !found {
		vmIDsByNamespace = set.NewStringSet()
		s.namespaceToUID[namespace] = vmIDsByNamespace
	}
	return vmIDsByNamespace
}

func (s *VirtualMachineStore) updateStatusOrCreateNoLock(updateInfo *VirtualMachineInfo) {
	prev, found := s.virtualMachines[updateInfo.UID]
	// This is needed in case of a race between the dispatchers
	if !found {
		s.addOrUpdateNoLock(updateInfo)
		return
	}
	// Remove previous VSOCK info
	s.removeVSOCKInfoNoLock(prev.UID, prev.VSOCKCID)
	// Update new VSOCK maps
	prev.VSOCKCID = s.addOrUpdateVSOCKInfoNoLock(updateInfo.UID, updateInfo.VSOCKCID)
	prev.Running = updateInfo.Running
}

func (s *VirtualMachineStore) addOrUpdateVSOCKInfoNoLock(uid string, vsockCID *uint32) *uint32 {
	if vsockCID == nil {
		return nil
	}
	s.uidToCID[uid] = *vsockCID
	s.cidToUID[*vsockCID] = uid
	// copy value before return
	val := *vsockCID
	return &val
}

func (s *VirtualMachineStore) removeVSOCKInfoNoLock(uid string, vsockCID *uint32) {
	if vsockCID == nil {
		return
	}
	delete(s.uidToCID, uid)
	delete(s.cidToUID, *vsockCID)
}

func (s *VirtualMachineStore) removeNoLock(uid string) {
	vm, found := s.virtualMachines[uid]
	if !found {
		return
	}
	log.Debugf("Removing virtual machine to store: name %q UID: %q", vm.Name, vm.UID)
	delete(s.virtualMachines, vm.UID)
	s.removeVSOCKInfoNoLock(vm.UID, vm.VSOCKCID)
	vmIDsByNamespace, found := s.namespaceToUID[vm.Namespace]
	if !found {
		log.Errorf("namespace %q was not found", vm.Namespace)
		return
	}
	vmIDsByNamespace.Remove(vm.UID)
	if len(vmIDsByNamespace) == 0 {
		delete(s.namespaceToUID, vm.Namespace)
	}
}

func (s *VirtualMachineStore) clearStatusNoLock(uid string) {
	vm, ok := s.virtualMachines[uid]
	if !ok {
		return
	}
	s.removeVSOCKInfoNoLock(vm.UID, vm.VSOCKCID)
	vm.VSOCKCID = nil
	// If the instance is removed the VirtualMachine will transition to Stopped
	vm.Running = false
}
