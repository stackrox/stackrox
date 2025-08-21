package store

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

const (
	vmUUID      = "vm-id"
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
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
			},
		},
		"running without VSOCK": {
			vm: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
		},
		"running with VSOCK": {
			vm: &VirtualMachineInfo{
				UID:       vmUUID,
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
			s.store.AddOrUpdateVirtualMachine(tCase.vm)
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
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
			},
		},
		"original running without VSOCK - update running with VSOCK": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"original running with VSOCK - update running with different VSOCK": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(2),
			},
		},
		"original running with VSOCK - update running with same VSOCK": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"original running with VSOCK - update running without VSOCK": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
		},
		"original running with VSOCK - update not running": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
				VSOCKCID:  nil,
			},
		},
		"original running without VSOCK - update not running": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
				VSOCKCID:  nil,
			},
		},
		"original nil - update running without vsock": {
			original: nil,
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
		},
		"original nil - update running with vsock": {
			original: nil,
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
		},
		"original nil - update not running": {
			original: nil,
			new: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   false,
				VSOCKCID:  nil,
			},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdateVirtualMachine(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.AddOrUpdateVirtualMachine(tCase.new)
			if tCase.original == nil && tCase.new == nil {
				s.assertEmpty()
				return
			} else {
				s.assertVM(tCase.new)
			}
			s.store.lock.Lock()
			defer s.store.lock.Unlock()
			s.Assert().Len(s.store.virtualMachines, 1)
			s.Assert().Len(s.store.namespaceToUID, 1)
			nsIOs, ok := s.store.namespaceToUID[tCase.new.Namespace]
			s.Assert().True(ok)
			s.Assert().Len(nsIOs, 1)
			if tCase.new.VSOCKCID == nil {
				s.Assert().Len(s.store.cidToUID, 0)
				s.Assert().Len(s.store.uidToCID, 0)
			} else {
				s.Assert().Len(s.store.cidToUID, 1)
				s.Assert().Len(s.store.uidToCID, 1)
			}
		})
	}
}

func (s *storeSuite) Test_AddVirtualMachineInstance() {
	vsockCID1 := newVSOCKCID(1)
	vsockCID2 := newVSOCKCID(2)
	cases := map[string]struct {
		original  *VirtualMachineInfo
		uid       string
		namespace string
		vsock     *uint32
		isRunning bool
		expected  *VirtualMachineInfo
	}{
		"original not running - instance running with vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
			},
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     newVSOCKCID(1),
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
		},
		"original not running - instance running without vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
			},
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     nil,
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
		},
		"original running with vsock - instance running without vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     nil,
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  nil,
				Running:   true,
			},
		},
		"original running with vsock - instance running with different vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     vsockCID2,
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID2,
				Running:   true,
			},
		},
		"original running with vsock - instance running with same vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  newVSOCKCID(1),
				Running:   true,
			},
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     vsockCID1,
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID1,
				Running:   true,
			},
		},
		"no virtual machine info - instance running with vsock": {
			original:  nil,
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     vsockCID1,
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Namespace: vmNamespace,
				VSOCKCID:  vsockCID1,
				Running:   true,
			},
		},
		"no virtual machine info - instance running without vsock": {
			original:  nil,
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     nil,
			isRunning: true,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Running:   true,
				Namespace: vmNamespace,
			},
		},
		"no virtual machine info - instance not running": {
			original:  nil,
			uid:       vmUUID,
			namespace: vmNamespace,
			vsock:     nil,
			isRunning: false,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Namespace: vmNamespace,
			},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdateVirtualMachine(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.AddOrUpdateVirtualMachineInstance(tCase.uid, tCase.namespace, tCase.vsock, tCase.isRunning)
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
		original    *VirtualMachineInfo
		uidToRemove string
		expected    *VirtualMachineInfo
	}{
		"original not running": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			uidToRemove: vmUUID,
			expected:    nil,
		},
		"original running with vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
			uidToRemove: vmUUID,
			expected:    nil,
		},
		"original running without vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
			uidToRemove: vmUUID,
			expected:    nil,
		},
		"original not running - remote no hit": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			uidToRemove: "other-uid",
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running with vsock - remove no hit": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
			uidToRemove: "other-uid",
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
		},
		"original running without vsock - remove no hit": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
			uidToRemove: "other-uid",
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  nil,
			},
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdateVirtualMachine(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.RemoveVirtualMachine(tCase.uidToRemove)
			if tCase.expected == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.expected)
			}
		})
	}
}

func (s *storeSuite) Test_RemoveVirtualMachineInstance() {
	vsockCID1 := newVSOCKCID(1)
	cases := map[string]struct {
		original *VirtualMachineInfo
		uid      string
		expected *VirtualMachineInfo
	}{
		"original not running": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			uid: vmUUID,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running with vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  newVSOCKCID(1),
			},
			uid: vmUUID,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running without vsock": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			uid: vmUUID,
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original not running - remove no hit": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
			uid: "other-uid",
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
			},
		},
		"original running with vsock - remove no hit": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
			uid: "other-uid",
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
				VSOCKCID:  vsockCID1,
			},
		},
		"original running without vsock - remove no hit": {
			original: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
			uid: "other-uid",
			expected: &VirtualMachineInfo{
				UID:       vmUUID,
				Name:      vmName,
				Namespace: vmNamespace,
				Running:   true,
			},
		},
		"no original": {
			original: nil,
			uid:      vmUUID,
			expected: nil,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			s.store.AddOrUpdateVirtualMachine(tCase.original)
			if tCase.original == nil {
				s.assertEmpty()
			} else {
				s.assertVM(tCase.original)
			}
			s.store.RemoveVirtualMachineInstance(tCase.uid)
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
			UID:       vmUUID,
			Name:      vmName,
			Namespace: vmNamespace,
		},
		{
			UID:       "other-uid-2",
			Name:      "other-name-2",
			Namespace: vmNamespace,
		},
		{
			UID:       "other-uid-3",
			Name:      "other-name-3",
			Namespace: "other-namespace",
		},
	}
	s.store = NewVirtualMachineStore()
	for _, vm := range vms {
		s.store.AddOrUpdateVirtualMachine(vm)
		s.assertVM(vm)
	}
	s.store.Cleanup()
	s.assertEmpty()
}

func (s *storeSuite) Test_OnNamespaceDeleted() {
	vms := []*VirtualMachineInfo{
		{
			UID:       vmUUID,
			Name:      vmName,
			Namespace: vmNamespace,
		},
		{
			UID:       "other-uid-2",
			Name:      "other-name-2",
			Namespace: vmNamespace,
		},
		{
			UID:       "other-uid-3",
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
					UID:       "other-uid-3",
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
				s.store.AddOrUpdateVirtualMachine(vm)
				s.assertVM(vm)
			}
			s.store.OnNamespaceDeleted(tCase.namespace)
			for _, vm := range tCase.expectedVMs {
				s.assertVM(vm)
			}
			s.store.lock.Lock()
			defer s.store.lock.Unlock()
			s.Assert().Len(s.store.virtualMachines, len(tCase.expectedVMs))
			s.Assert().Nil(s.store.namespaceToUID[tCase.namespace])
		})
	}
}

func (s *storeSuite) Test_GetVirtualMachine() {
	vm := &VirtualMachineInfo{
		UID:       vmUUID,
		Name:      vmName,
		Namespace: vmNamespace,
		VSOCKCID:  newVSOCKCID(1),
		Running:   true,
	}
	s.Run("success", func() {
		s.store.AddOrUpdateVirtualMachine(vm)
		s.assertVM(vm)
		actual := s.store.Get(vmUUID)
		s.Assert().NotNil(actual)
		assertVMs(s.T(), vm, actual)
	})
	s.Run("no hit", func() {
		s.store.AddOrUpdateVirtualMachine(vm)
		s.assertVM(vm)
		actual := s.store.Get("other uid")
		s.Assert().Nil(actual)
	})
}

func (s *storeSuite) assertEmpty() {
	s.store.lock.Lock()
	defer s.store.lock.Unlock()
	s.Assert().Len(s.store.virtualMachines, 0)
	s.Assert().Len(s.store.namespaceToUID, 0)
	s.Assert().Len(s.store.cidToUID, 0)
	s.Assert().Len(s.store.uidToCID, 0)
}

func (s *storeSuite) assertVM(expected *VirtualMachineInfo) {
	s.store.lock.Lock()
	defer s.store.lock.Unlock()
	actual, ok := s.store.virtualMachines[expected.UID]
	s.Assert().True(ok)
	s.Assert().Equal(expected.UID, actual.UID)
	s.Assert().Equal(expected.Name, actual.Name)
	s.Assert().Equal(expected.Namespace, actual.Namespace)
	s.Assert().Equal(expected.VSOCKCID, actual.VSOCKCID)
	s.Assert().Equal(expected.Running, actual.Running)
	nsIDs, ok := s.store.namespaceToUID[expected.Namespace]
	s.Assert().True(ok)
	s.Assert().Contains(nsIDs, expected.UID)
	if expected.VSOCKCID == nil {
		return
	}
	cid, ok := s.store.uidToCID[expected.UID]
	s.Assert().True(ok)
	s.Assert().Equal(*expected.VSOCKCID, cid)
	uid, ok := s.store.cidToUID[*expected.VSOCKCID]
	s.Assert().True(ok)
	s.Assert().Equal(expected.UID, uid)
}

func assertVMs(t *testing.T, expected *VirtualMachineInfo, actual *VirtualMachineInfo) {
	assert.Equal(t, expected.UID, actual.UID)
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
