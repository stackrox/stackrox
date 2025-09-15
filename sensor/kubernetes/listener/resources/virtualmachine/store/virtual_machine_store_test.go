package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	vmID        = "vm-id"
	vmName      = "vm-name"
	vmNamespace = "vm-namespace"
)

func TestVirtualMachineStore(t *testing.T) {
	suite.Run(t, new(storeSuite))
}

type storeSuite struct {
	suite.Suite
	store *VirtualMachineStore
}

func (s *storeSuite) SetupSubTest() {
	s.store = NewVirtualMachineStore()
}

var _ suite.SetupSubTest = (*storeSuite)(nil)

func (s *storeSuite) Test_AddVirtualMachine() {
	cases := map[string]struct {
		vm *VirtualMachineInfo
	}{
		"not running": {
			vm: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
			},
		},
		"running without VSOCK": {
			vm: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
		},
		"running with VSOCK": {
			vm: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"nil": {
			vm: nil,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdate(tCase.vm)
			if tCase.vm == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.vm)
			}
		})
	}
}

func (s *storeSuite) Test_UpdateVirtualMachine() {
	cases := map[string]struct {
		original *VirtualMachineInfo
		new      *VirtualMachineInfo
	}{
		"original not running - update running": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
			},
		},
		"original running without VSOCK - update running with VSOCK": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"original running with VSOCK - update running with different VSOCK": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(2),
			},
		},
		"original running with VSOCK - update running with same VSOCK": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"original running with VSOCK - update running without VSOCK": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
		},
		"original running with VSOCK - update not running": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
				VSOCKCID:  nil,
			},
		},
		"original running without VSOCK - update not running": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
				VSOCKCID:  nil,
			},
		},
		"original nil - update running without vsock": {
			original: nil,
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
		},
		"original nil - update running with vsock": {
			original: nil,
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"original nil - update not running": {
			original: nil,
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
				VSOCKCID:  nil,
			},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdate(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.AddOrUpdate(tCase.new)
			if tCase.original == nil && tCase.new == nil {
				s.assertEmpty()
				return
			} else {
				s.assertVM(tCase.new)
			}
			s.store.lock.Lock()
			defer s.store.lock.Unlock()
			s.Assert().Len(s.store.virtualMachines, 1)
			s.Assert().Len(s.store.namespaceToID, 1)
			nsIOs, ok := s.store.namespaceToID[tCase.new.Namespace]
			s.Assert().True(ok)
			s.Assert().Len(nsIOs, 1)
			if tCase.new.VSOCKCID == nil {
				s.Assert().Len(s.store.cidToID, 0)
				s.Assert().Len(s.store.idToCID, 0)
			} else {
				s.Assert().Len(s.store.cidToID, 1)
				s.Assert().Len(s.store.idToCID, 1)
			}
		})
	}
}

func (s *storeSuite) Test_UpdateStateOrCreate() {
	vsockCID1 := newVSOCKCID(1)
	vsockCID2 := newVSOCKCID(2)
	cases := map[string]struct {
		original *VirtualMachineInfo
		new      *VirtualMachineInfo
		expected *VirtualMachineInfo
	}{
		"original not running - instance running with vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
		},
		"original not running - instance running without vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
		},
		"original running with vsock - instance running without vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
		},
		"original running with vsock - instance running with different vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID2,
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID2,
				Running:   true,
			},
		},
		"original running with vsock - instance running with same vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID1,
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID1,
				Running:   true,
			},
		},
		"no virtual machine info - instance running with vsock": {
			original: nil,
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID1,
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID1,
				Running:   true,
			},
		},
		"no virtual machine info - instance running without vsock": {
			original: nil,
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Running:   true,
				Namespace: vmNamespace,
			},
		},
		"no virtual machine info - instance not running": {
			original: nil,
			new: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   false,
			},
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdate(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.UpdateStateOrCreate(tCase.new)
			if tCase.expected == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.expected)
			}
		})
	}
}

func (s *storeSuite) Test_RemoveVirtualMachine() {
	vsockCID1 := newVSOCKCID(1)
	cases := map[string]struct {
		original   *VirtualMachineInfo
		idToRemove VMID
		expected   *VirtualMachineInfo
	}{
		"original not running": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			idToRemove: vmID,
			expected:   nil,
		},
		"original running with vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
			idToRemove: vmID,
			expected:   nil,
		},
		"original running without vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
			idToRemove: vmID,
			expected:   nil,
		},
		"original not running - remote no hit": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			idToRemove: "other-id",
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running with vsock - remove no hit": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
			idToRemove: "other-id",
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
		},
		"original running without vsock - remove no hit": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
			idToRemove: "other-id",
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdate(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.Remove(tCase.idToRemove)
			if tCase.expected == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.expected)
			}
		})
	}
}

func (s *storeSuite) Test_ClearState() {
	vsockCID1 := newVSOCKCID(1)
	cases := map[string]struct {
		original *VirtualMachineInfo
		id       VMID
		expected *VirtualMachineInfo
	}{
		"original not running": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			id: vmID,
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running with vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			id: vmID,
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running without vsock": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			id: vmID,
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original not running - remove no hit": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			id: "other-id",
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running with vsock - remove no hit": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
			id: "other-id",
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
		},
		"original running without vsock - remove no hit": {
			original: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			id: "other-id",
			expected: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
		},
		"no original": {
			original: nil,
			id:       vmID,
			expected: nil,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdate(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.ClearState(tCase.id)
			if tCase.expected == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.expected)
			}
		})
	}
}

func (s *storeSuite) Test_Cleanup() {
	vms := []*VirtualMachineInfo{
		{
			ID:        vmID,
			Name:      vmName,
			Namespace: vmNamespace,
		},
		{
			ID:        "other-id-2",
			Name:      "other-name-2",
			Namespace: vmNamespace,
		},
		{
			ID:        "other-id-3",
			Name:      "other-name-3",
			Namespace: "other-namespace",
		},
	}
	s.store = NewVirtualMachineStore()
	for _, vm := range vms {
		s.store.AddOrUpdate(vm)
		s.assertVM(vm)
	}
	s.store.Cleanup()
	s.assertEmpty()
}

func (s *storeSuite) Test_OnNamespaceDeleted() {
	vms := []*VirtualMachineInfo{
		{
			ID:        vmID,
			Name:      vmName,
			Namespace: vmNamespace,
		},
		{
			ID:        "other-id-2",
			Name:      "other-name-2",
			Namespace: vmNamespace,
		},
		{
			ID:        "other-id-3",
			Name:      "other-name-3",
			Namespace: "other-namespace",
		},
	}
	cases := map[string]struct {
		vms         []*VirtualMachineInfo
		namespace   string
		expectedVMs []*VirtualMachineInfo
	}{
		"remove namespace": {
			vms:       vms,
			namespace: vmNamespace,
			expectedVMs: []*VirtualMachineInfo{
				{
					ID:        "other-id-3",
					Name:      "other-name-3",
					Namespace: "other-namespace",
				},
			},
		},
		"remove namespace no hit": {
			vms:         vms,
			namespace:   "no-hit-namespace",
			expectedVMs: vms,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			for _, vm := range tCase.vms {
				s.store.AddOrUpdate(vm)
				s.assertVM(vm)
			}
			s.store.OnNamespaceDeleted(tCase.namespace)
			for _, vm := range tCase.expectedVMs {
				s.assertVM(vm)
			}
			s.store.lock.Lock()
			defer s.store.lock.Unlock()
			s.Assert().Len(s.store.virtualMachines, len(tCase.expectedVMs))
			s.Assert().Nil(s.store.namespaceToID[tCase.namespace])
		})
	}
}

func (s *storeSuite) Test_GetVirtualMachine() {
	vm := &VirtualMachineInfo{
		ID:        vmID,
		Name:      vmName,
		Namespace: vmNamespace,
		VSOCKCID:  newVSOCKCID(1),
		Running:   true,
	}
	s.Run("success", func() {
		s.store.AddOrUpdate(vm)
		s.assertVM(vm)
		actual := s.store.Get(vmID)
		s.Assert().NotNil(actual)
		assertVMs(s.T(), vm, actual)
	})
	s.Run("no hit", func() {
		s.store.AddOrUpdate(vm)
		s.assertVM(vm)
		actual := s.store.Get("other id")
		s.Assert().Nil(actual)
	})
}

func (s *storeSuite) Test_HasVirtualMachine() {
	vm := &VirtualMachineInfo{
		ID:        vmID,
		Name:      vmName,
		Namespace: vmNamespace,
		VSOCKCID:  newVSOCKCID(1),
		Running:   true,
	}
	s.Run("success", func() {
		s.store.AddOrUpdate(vm)
		s.assertVM(vm)
		s.Assert().True(s.store.Has(vmID))
	})
	s.Run("no hit", func() {
		s.store.AddOrUpdate(vm)
		s.assertVM(vm)
		s.Assert().False(s.store.Has("other id"))
	})
}

func (s *storeSuite) Test_GetVirtualMachineFromCID() {
	cases := map[string]struct {
		vm         *VirtualMachineInfo
		cid        uint32
		expectedVM *VirtualMachineInfo
	}{
		"should find a valid CID": {
			vm: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			cid: 1,
			expectedVM: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
		},
		"should return nil an invalid CID": {
			vm: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			cid:        2, // Invalid CID
			expectedVM: nil,
		},
		"should return nil if the VM does not have a Vsock CID yet": {
			vm: &VirtualMachineInfo{
				ID:        vmID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
			cid:        1, // VM does not have a cid
			expectedVM: nil,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdate(tCase.vm)
			s.assertVM(tCase.vm)
			actual := s.store.GetFromCID(tCase.cid)
			if tCase.expectedVM == nil {
				s.Assert().Nil(actual)
			} else {
				assertVMs(s.T(), tCase.expectedVM, actual)
			}
		})
	}
}

func (s *storeSuite) assertEmpty() {
	s.store.lock.Lock()
	defer s.store.lock.Unlock()
	s.Assert().Len(s.store.virtualMachines, 0)
	s.Assert().Len(s.store.namespaceToID, 0)
	s.Assert().Len(s.store.cidToID, 0)
	s.Assert().Len(s.store.idToCID, 0)
}

func (s *storeSuite) assertVM(expected *VirtualMachineInfo) {
	s.store.lock.Lock()
	defer s.store.lock.Unlock()
	actual, ok := s.store.virtualMachines[expected.ID]
	s.Assert().True(ok)
	s.Assert().Equal(expected.ID, actual.ID)
	s.Assert().Equal(expected.Name, actual.Name)
	s.Assert().Equal(expected.Namespace, actual.Namespace)
	s.Assert().Equal(expected.VSOCKCID, actual.VSOCKCID)
	s.Assert().Equal(expected.Running, actual.Running)
	nsIDs, ok := s.store.namespaceToID[expected.Namespace]
	s.Assert().True(ok)
	s.Assert().Contains(nsIDs, expected.ID)
	if expected.VSOCKCID == nil {
		return
	}
	cid, ok := s.store.idToCID[expected.ID]
	s.Assert().True(ok)
	s.Assert().Equal(*expected.VSOCKCID, cid)
	id, ok := s.store.cidToID[*expected.VSOCKCID]
	s.Assert().True(ok)
	s.Assert().Equal(expected.ID, id)
}

func assertVMs(t *testing.T, expected *VirtualMachineInfo, actual *VirtualMachineInfo) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Name, actual.Name)
	assert.Equal(t, expected.Namespace, actual.Namespace)
	assert.Equal(t, expected.Running, actual.Running)
	if expected.VSOCKCID == nil {
		assert.Nil(t, actual.VSOCKCID)
	} else {
		assert.Equal(t, *expected.VSOCKCID, *actual.VSOCKCID)
	}
}

func newVSOCKCID(val uint32) *uint32 {
	return &val
}
