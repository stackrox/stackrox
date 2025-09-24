package dispatcher

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/pkg/virtualmachine"
	vmInfo "github.com/stackrox/rox/sensor/common/virtualmachine"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/dispatcher/mocks"
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
	s.mockCtrl = gomock.NewController(s.T())
	s.store = mocks.NewMockvirtualMachineStore(s.mockCtrl)
	s.dispatcher = NewVirtualMachineInstanceDispatcher(clusterID, s.store)
}

func (s *virtualMachineInstanceSuite) TearDownSubTest() {
	s.mockCtrl.Finish()
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
				s.store.EXPECT().AddOrUpdate(gomock.Eq(
					&vmInfo.Info{
						ID:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
				).Times(1)
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
					},
				},
			}),
		},
		"no VirtualMachine owner reference update resource": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().AddOrUpdate(gomock.Eq(
					&vmInfo.Info{
						ID:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
				).Times(1)
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
					},
				},
			}),
		},
		"no VirtualMachine owner reference remove resource": {
			action: central.ResourceAction_REMOVE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().Remove(gomock.Eq(vmInfo.VMID(vmiUID))).Times(1)
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
					},
				},
			}),
		},
		"no VirtualMachine owner reference sync resource": {
			action: central.ResourceAction_SYNC_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstanceWithOwnerKind(vmiUID, vmiName, vmiNamespace, ownerUID, "Not-VirtualMachine", nil, v1.Scheduled)),
			expectFn: func() {
				s.store.EXPECT().AddOrUpdate(gomock.Eq(
					&vmInfo.Info{
						ID:        vmiUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
				).Times(1)
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
					},
				},
			}),
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
	return newVirtualMachineInstanceWithOwnerKind(uid, name, namespace, owner, virtualmachine.VirtualMachine.Kind, vsock, phase)
}

func toUnstructured(obj any) *unstructured.Unstructured {
	ret := &unstructured.Unstructured{}
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	tmp, _ := json.Marshal(unstructuredObj)
	_ = ret.UnmarshalJSON(tmp)
	return ret
}
