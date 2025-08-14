package resources

import (
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/sync"
	kubeVirtV1 "kubevirt.io/api/core/v1"
)

type virtualMachineWrap struct {
	vm       *kubeVirtV1.VirtualMachine
	original interface{}
}

type VirtualMachineStore struct {
	lock sync.RWMutex

	virtualMachineIDs map[string]map[string]struct{}
	virtualMachines   map[string]*virtualMachineWrap
}

func newVirtualMachineStore() *VirtualMachineStore {
	return &VirtualMachineStore{
		virtualMachines:   make(map[string]*virtualMachineWrap),
		virtualMachineIDs: make(map[string]map[string]struct{}),
	}
}

func (s *VirtualMachineStore) addOrUpdateVirtualMachine(vmWrap *virtualMachineWrap) {
	log.Info("Pushing virtual machine to store", "name", vmWrap.vm.GetName(), "UID", vmWrap.vm.GetUID())
	concurrency.WithLock(&s.lock, func() {
		vmNamespace := vmWrap.vm.GetNamespace()
		vmIDsByNamespace, found := s.virtualMachineIDs[vmNamespace]
		if !found {
			vmIDsByNamespace = make(map[string]struct{})
			s.virtualMachineIDs[vmNamespace] = vmIDsByNamespace
		}
		vmID := string(vmWrap.vm.GetUID())
		s.virtualMachines[vmID] = vmWrap
		vmIDsByNamespace[vmID] = struct{}{}
	})
}

func (s *VirtualMachineStore) removeVirtualMachine(vmWrap *virtualMachineWrap) {
	log.Info("Removing virtual machine from store", "name", vmWrap.vm.GetName(), "UID", vmWrap.vm.GetUID())
	concurrency.WithLock(&s.lock, func() {
		vmID := string(vmWrap.vm.GetUID())
		vmNamespace := vmWrap.vm.GetNamespace()
		delete(s.virtualMachines, vmID)
		vmIDsByNamespace, found := s.virtualMachineIDs[vmNamespace]
		if !found {
			return
		}
		delete(vmIDsByNamespace, vmID)
	})
}

func (s *VirtualMachineStore) Cleanup() {
	concurrency.WithLock(&s.lock, func() {
		s.virtualMachines = make(map[string]*virtualMachineWrap)
		s.virtualMachineIDs = make(map[string]map[string]struct{})
	})
}

func (s *VirtualMachineStore) CountVirtualMachinesForNamespace(namespace string) int {
	return concurrency.WithRLock1(&s.lock, func() int {
		return len(s.virtualMachineIDs[namespace])
	})
}

func (s *VirtualMachineStore) OnNamespaceDeleted(namespace string) {
	concurrency.WithLock(&s.lock, func() {
		vmIDsByNamespace := s.virtualMachineIDs[namespace]
		for vmID := range vmIDsByNamespace {
			delete(s.virtualMachines, vmID)
		}
		delete(s.virtualMachineIDs, namespace)
	})
}

func (s *VirtualMachineStore) Get(id string) *kubeVirtV1.VirtualMachine {
	var rv *kubeVirtV1.VirtualMachine
	concurrency.WithRLock(&s.lock, func() {
		wrapper := s.virtualMachines[id]
		if wrapper == nil {
			return
		}
		rv = wrapper.vm.DeepCopy()
	})
	return rv
}
