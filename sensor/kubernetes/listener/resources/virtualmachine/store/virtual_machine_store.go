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

// AddOrUpdateVirtualMachine upserts a new VirtualMachine
func (s *VirtualMachineStore) AddOrUpdateVirtualMachine(vm *VirtualMachineInfo) {
	if vm == nil {
		return
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addOrUpdateVirtualMachineNoLock(vm)
}

// AddOrUpdateVirtualMachineInstance upserts a new VirtualMachineInstance
func (s *VirtualMachineStore) AddOrUpdateVirtualMachineInstance(uid, namespace string, vsockCID *uint32, isRunning bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.addOrUpdateVirtualMachineInstanceNoLock(uid, namespace, vsockCID, isRunning)
}

// RemoveVirtualMachine removes a VirtualMachine
func (s *VirtualMachineStore) RemoveVirtualMachine(uid string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeVirtualMachineNoLock(uid)
}

// RemoveVirtualMachineInstance removes a VirtualMachineInstance
func (s *VirtualMachineStore) RemoveVirtualMachineInstance(uid string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.removeVirtualMachineInstanceNoLock(uid)
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
		s.removeVirtualMachineNoLock(vmID)
	}
}

// Get returns the VirtualMachineInfo associated with the given UID
func (s *VirtualMachineStore) Get(uid string) *VirtualMachineInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.virtualMachines[uid].Copy()
}

func (s *VirtualMachineStore) addOrUpdateVirtualMachineNoLock(vm *VirtualMachineInfo) {
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
		s.addOrUpdateVSOCKInfoNoLock(vm.UID, vm.VSOCKCID)
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

func (s *VirtualMachineStore) addOrUpdateVirtualMachineInstanceNoLock(uid, namespace string, vsockCID *uint32, isRunning bool) {
	var vm *VirtualMachineInfo
	vm, found := s.virtualMachines[uid]
	// This is needed in case of a race between the dispatchers
	if !found {
		vm = &VirtualMachineInfo{
			UID:       uid,
			Namespace: namespace,
		}
		vmIDsByNamespace := s.getOrCreateNamespaceSet(vm.Namespace)
		vmIDsByNamespace.Add(vm.UID)
		s.virtualMachines[uid] = vm

	}
	// Remove previous VSOCK info
	s.removeVSOCKInfoNoLock(uid, vm.VSOCKCID)
	vm.VSOCKCID = nil
	// Update new VSOCK maps
	s.addOrUpdateVSOCKInfoNoLock(uid, vsockCID)
	// Copy the VSOCK info
	if vsockCID != nil {
		val := *vsockCID
		vm.VSOCKCID = &val
	}
	vm.Running = isRunning
}

func (s *VirtualMachineStore) addOrUpdateVSOCKInfoNoLock(uid string, vsockCID *uint32) {
	if vsockCID == nil {
		return
	}
	s.uidToCID[uid] = *vsockCID
	s.cidToUID[*vsockCID] = uid
}

func (s *VirtualMachineStore) removeVSOCKInfoNoLock(uid string, vsockCID *uint32) {
	if vsockCID == nil {
		return
	}
	delete(s.uidToCID, uid)
	delete(s.cidToUID, *vsockCID)
}

func (s *VirtualMachineStore) removeVirtualMachineNoLock(uid string) {
	vm, found := s.virtualMachines[uid]
	if !found {
		return
	}
	log.Debugf("Removing virtual machine to store: name %q UID: %q", vm.Name, vm.UID)
	delete(s.virtualMachines, vm.UID)
	vmIDsByNamespace, found := s.namespaceToUID[vm.Namespace]
	if found {
		vmIDsByNamespace.Remove(vm.UID)
		if len(vmIDsByNamespace) == 0 {
			delete(s.namespaceToUID, vm.Namespace)
		}
	}
	s.removeVSOCKInfoNoLock(vm.UID, vm.VSOCKCID)
}

func (s *VirtualMachineStore) removeVirtualMachineInstanceNoLock(uid string) {
	vm, ok := s.virtualMachines[uid]
	if !ok {
		return
	}
	s.removeVSOCKInfoNoLock(vm.UID, vm.VSOCKCID)
	vm.VSOCKCID = nil
	// If the instance is removed the VirtualMachine will transition to Stopped
	vm.Running = false
}
