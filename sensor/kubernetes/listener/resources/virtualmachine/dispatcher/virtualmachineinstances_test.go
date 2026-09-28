package dispatcher

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/features"
	pkgVM "github.com/stackrox/rox/pkg/virtualmachine"
	"github.com/stackrox/rox/sensor/common/centralcaps"
	vmInfo "github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/dispatcher/mocks"
	vmstore "github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

const (
	clusterID    = "cluster-id"
	ownerUID     = "vm-id"
	vmiUID       = "vmi-id"
	vmiName      = "vmi-name"
	vmiNamespace = "vmi-namespace"
)

func TestVirtualMachineInstancesDispatcher(t *testing.T) {
	suite.Run(t, new(virtualMachineInstanceSuite))
}

type virtualMachineInstanceSuite struct {
	suite.Suite
	mockCtrl   *gomock.Controller
	store      *mocks.MockvirtualMachineStore
	dispatcher *VirtualMachineInstanceDispatcher
}

var _ suite.SetupSubTest = (*virtualMachineInstanceSuite)(nil)
var _ suite.TearDownSubTest = (*virtualMachineInstanceSuite)(nil)

func (s *virtualMachineInstanceSuite) SetupSubTest() {
	s.T().Setenv(features.VirtualMachines.EnvVar(), "true")
	centralcaps.Set([]centralsensor.CentralCapability{centralsensor.VirtualMachinesSupported})

	s.mockCtrl = gomock.NewController(s.T())
	s.store = mocks.NewMockvirtualMachineStore(s.mockCtrl)
	s.dispatcher = NewVirtualMachineInstanceDispatcher(clusterID, s.store)
}

func (s *virtualMachineInstanceSuite) TearDownSubTest() {
	s.mockCtrl.Finish()
	// Reset global capability state to prevent test pollution.
	// If not reset, subsequent tests may incorrectly assume capabilities are present/absent,
	// leading to false negatives that could hide regressions.
	centralcaps.Set(nil)
}

func (s *virtualMachineInstanceSuite) Test_VirtualMachineInstanceEvents() {
	var vsockVal uint32 = 1
	cases := map[string]struct {
		action      central.ResourceAction
		obj         any
		expectFn    func()
		expectedMsg *component.ResourceEvent
	}{
		"sync event": {
			action: central.ResourceAction_SYNC_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(true),
					s.store.EXPECT().UpdateStateOrCreate(
						gomock.Eq(&vmInfo.Info{
							ID:        ownerUID,
							Name:      vmiName,
							Namespace: vmiNamespace,
							VSOCKCID:  nil,
							Running:   false,
						}),
					).Times(1),
				)
			},
			expectedMsg: nil,
		},
		"create event": {
			action: central.ResourceAction_CREATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(true),
					s.store.EXPECT().UpdateStateOrCreate(
						gomock.Eq(&vmInfo.Info{
							ID:        ownerUID,
							Name:      vmiName,
							Namespace: vmiNamespace,
							VSOCKCID:  nil,
							Running:   false,
						}),
					).Times(1),
					s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(nil),
				)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     ownerUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"update event": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(true),
					s.store.EXPECT().UpdateStateOrCreate(gomock.Eq(
						&vmInfo.Info{
							ID:        ownerUID,
							Name:      vmiName,
							Namespace: vmiNamespace,
							VSOCKCID:  nil,
							Running:   false,
						}),
					).Times(1),
					s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(nil),
				)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     ownerUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"remove event": {
			action: central.ResourceAction_REMOVE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(true),
					s.store.EXPECT().ClearState(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1),
					s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(nil),
				)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     ownerUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"remove event keeps stored agent facts": {
			action: central.ResourceAction_REMOVE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(true),
					s.store.EXPECT().ClearState(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1),
					s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(&vmInfo.Info{
						ID: ownerUID,
						AgentFacts: map[string]string{
							pkgVM.DetectedGuestOSKey:   "Red Hat Enterprise Linux 9.8",
							pkgVM.ActivationStatusKey:  pkgVM.ActivationStatusActive,
							pkgVM.DNFMetadataStatusKey: pkgVM.DNFMetadataStatusAvailable,
							pkgVM.AgentVersionKey:      "development",
						},
					}),
				)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     ownerUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts: map[string]string{
							pkgVM.GuestOSKey:           pkgVM.UnknownGuestOS,
							pkgVM.DetectedGuestOSKey:   "Red Hat Enterprise Linux 9.8",
							pkgVM.ActivationStatusKey:  pkgVM.ActivationStatusActive,
							pkgVM.DNFMetadataStatusKey: pkgVM.DNFMetadataStatusAvailable,
							pkgVM.AgentVersionKey:      "development",
						},
					},
				},
			}),
		},
		"no unstructured object": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled),
			expectFn:    func() {},
			expectedMsg: nil,
		},
		"no virtual machine instance": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         toUnstructured(&v1.VirtualMachine{}),
			expectFn:    func() {},
			expectedMsg: nil,
		},
		"no VirtualMachine owner reference create resource": {
			action: central.ResourceAction_CREATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().UpdateStateOrCreate(gomock.Eq(
					&vmInfo.Info{
						ID:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
				).Times(1)
				s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(vmiUID))).Times(1).Return(nil)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmiUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"no VirtualMachine owner reference update resource": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().UpdateStateOrCreate(gomock.Eq(
					&vmInfo.Info{
						ID:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
				).Times(1)
				s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(vmiUID))).Times(1).Return(nil)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmiUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"no VirtualMachine owner reference remove resource": {
			action: central.ResourceAction_REMOVE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().Remove(gomock.Eq(vmInfo.VMID(vmiUID))).Times(1)
				s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(vmiUID))).Times(1).Return(nil)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmiUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"no VirtualMachine owner reference sync resource": {
			action: central.ResourceAction_SYNC_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().UpdateStateOrCreate(gomock.Eq(
					&vmInfo.Info{
						ID:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
				).Times(1)
				s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(vmiUID))).Times(1).Return(nil)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmiUID,
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						ClusterId: clusterID,
						State:     virtualMachineV1.VirtualMachine_STOPPED,
						Facts:     getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"call to store Has returns false": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(false),
					s.store.EXPECT().UpdateStateOrCreate(gomock.Eq(
						&vmInfo.Info{
							ID:        ownerUID,
							Name:      vmiName,
							Namespace: vmiNamespace,
							VSOCKCID:  nil,
							Running:   false,
						}),
					).Times(1),
				)
			},
			expectedMsg: nil,
		},
		"update state of virtual machine": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, &vsockVal, v1.Running)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Has(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(true),
					s.store.EXPECT().UpdateStateOrCreate(gomock.Eq(
						&vmInfo.Info{
							ID:        ownerUID,
							Name:      vmiName,
							Namespace: vmiNamespace,
							VSOCKCID:  &vsockVal,
							Running:   true,
						}),
					).Times(1),
					s.store.EXPECT().Get(gomock.Eq(vmInfo.VMID(ownerUID))).Times(1).Return(nil),
				)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     ownerUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:          ownerUID,
						Name:        vmiName,
						Namespace:   vmiNamespace,
						ClusterId:   clusterID,
						VsockCid:    int32(vsockVal),
						VsockCidSet: true,
						State:       virtualMachineV1.VirtualMachine_RUNNING,
						Facts:       getFactsForTest(s.T(), pkgVM.UnknownGuestOS),
					},
				},
			}),
		},
		"capability not supported": {
			action: central.ResourceAction_CREATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				// Guarded path must not emit an event nor touch the store.
				s.store.EXPECT().Has(gomock.Any()).Times(0)
				s.store.EXPECT().UpdateStateOrCreate(gomock.Any()).Times(0)
				centralcaps.Set(nil)
			},
			expectedMsg: nil,
		},
		"feature flag disabled": {
			action: central.ResourceAction_CREATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				// Guarded path must not emit an event nor touch the store.
				s.store.EXPECT().Has(gomock.Any()).Times(0)
				s.store.EXPECT().UpdateStateOrCreate(gomock.Any()).Times(0)
				s.T().Setenv(features.VirtualMachines.EnvVar(), "false")
			},
			expectedMsg: nil,
		},
	}
	for tName, tCase := range cases {
		s.Run(tName, func() {
			tCase.expectFn()
			actual := s.dispatcher.ProcessEvent(tCase.obj, nil, tCase.action)
			if tCase.expectedMsg != nil {
				s.Require().NotNil(actual)
				s.Require().Len(actual.ForwardMessages, 1)
				s.Assert().True(proto.Equal(tCase.expectedMsg.ForwardMessages[0], actual.ForwardMessages[0]))
			} else {
				s.Assert().Nil(actual)
			}
		})
	}
}

// Test_OwnerlessVMIScheduledThenRunningIsScrapable covers an ownerless VMI first
// seen before Running: later Running must reach the store and ListRunning.
func (s *virtualMachineInstanceSuite) Test_OwnerlessVMIScheduledThenRunningIsScrapable() {
	cases := map[string]struct {
		buildVMI func(phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance
	}{
		"should mark ownerless VMI running after Scheduled then Running": {
			buildVMI: func(phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
				vmi := newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, "", "", nil, phase)
				vmi.OwnerReferences = nil
				return vmi
			},
		},
		"should mark ReplicaSet-owned VMI running after Scheduled then Running": {
			buildVMI: func(phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
				return newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "VirtualMachineInstanceReplicaSet", nil, phase)
			},
		},
	}
	for name, tc := range cases {
		s.Run(name, func() {
			realStore := vmstore.NewVirtualMachineStore()
			d := NewVirtualMachineInstanceDispatcher(clusterID, realStore)

			scheduled := d.ProcessEvent(toUnstructured(tc.buildVMI(v1.Scheduled)), nil, central.ResourceAction_CREATE_RESOURCE)
			s.Require().NotNil(scheduled)
			stored := realStore.Get(vmInfo.VMID(vmiUID))
			s.Require().NotNil(stored)
			s.False(stored.Running)
			s.Empty(realStore.ListRunning())

			running := d.ProcessEvent(toUnstructured(tc.buildVMI(v1.Running)), nil, central.ResourceAction_UPDATE_RESOURCE)
			s.Require().NotNil(running)
			s.Require().Len(running.ForwardMessages, 1)
			vmEvent := running.ForwardMessages[0].GetVirtualMachine()
			s.Require().NotNil(vmEvent)
			s.Equal(virtualMachineV1.VirtualMachine_RUNNING, vmEvent.GetState())

			stored = realStore.Get(vmInfo.VMID(vmiUID))
			s.Require().NotNil(stored)
			s.True(stored.Running)
			runningList := realStore.ListRunning()
			s.Require().Len(runningList, 1)
			s.Equal(vmInfo.VMID(vmiUID), runningList[0].ID)
		})
	}
}

func newVirtualMachineInstanceWithOwnerKind(uid, name, namespace, owner, kind string, vsock *uint32, phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID(uid),
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					UID:  types.UID(owner),
					Kind: kind,
					Name: name,
				},
			},
		},
		Status: v1.VirtualMachineInstanceStatus{
			Phase:    phase,
			VSOCKCID: vsock,
		},
	}
}

func newVirtualMachineInstance(uid, name, namespace, owner string, vsock *uint32, phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
	return newVirtualMachineInstanceWithOwnerKind(uid, name, namespace, owner, pkgVM.VirtualMachine.Kind, vsock, phase)
}

func toUnstructured(obj any) *unstructured.Unstructured {
	ret := &unstructured.Unstructured{}
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	tmp, _ := json.Marshal(unstructuredObj)
	_ = ret.UnmarshalJSON(tmp)
	return ret
}
