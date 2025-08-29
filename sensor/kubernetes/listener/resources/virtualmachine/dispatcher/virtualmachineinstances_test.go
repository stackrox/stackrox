package dispatcher

import (
	"encoding/json"
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	virtualMachineV1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/sensor/kubernetes/eventpipeline/component"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/dispatcher/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/listener/resources/virtualmachine/store"
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
	vmiUUID      = "vmi-id"
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
	var nilPtr *uint32
	var vsockVal uint32 = 1
	cases := map[string]struct {
		action      central.ResourceAction
		obj         any
		expectFn    func()
		expectedMsg *component.ResourceEvent
	}{
		"sync event": {
			action: central.ResourceAction_SYNC_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
					s.store.EXPECT().AddOrUpdateVirtualMachineInstance(
						gomock.Eq(ownerUID),
						gomock.Eq(vmiNamespace),
						gomock.Eq(nilPtr),
						gomock.Eq(false)).Times(1),
				)
			},
			expectedMsg: nil,
		},
		"create event": {
			action: central.ResourceAction_CREATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
					s.store.EXPECT().AddOrUpdateVirtualMachineInstance(
						gomock.Eq(ownerUID),
						gomock.Eq(vmiNamespace),
						gomock.Eq(nilPtr),
						gomock.Eq(false)).Times(1),
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
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
					},
				},
			}),
		},
		"update event": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
					s.store.EXPECT().AddOrUpdateVirtualMachineInstance(
						gomock.Eq(ownerUID),
						gomock.Eq(vmiNamespace),
						gomock.Eq(nilPtr),
						gomock.Eq(false)).Times(1),
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
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
					},
				},
			}),
		},
		"remove event": {
			action: central.ResourceAction_REMOVE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
					s.store.EXPECT().RemoveVirtualMachineInstance(
						gomock.Eq(ownerUID)).Times(1),
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
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
					},
				},
			}),
		},
		"no unstructured object": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled),
			expectFn:    func() {},
			expectedMsg: nil,
		},
		"no virtual machine instance": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         toUnstructured(&v1.VirtualMachine{}),
			expectFn:    func() {},
			expectedMsg: nil,
		},
		"no owner reference": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         toUnstructured(&v1.VirtualMachineInstance{}),
			expectFn:    func() {},
			expectedMsg: nil,
		},
		"second call to store Get returns nil": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nil,
						Running:   false,
					}),
					s.store.EXPECT().AddOrUpdateVirtualMachineInstance(
						gomock.Eq(ownerUID),
						gomock.Eq(vmiNamespace),
						gomock.Eq(nilPtr),
						gomock.Eq(false)).Times(1),
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(nil),
				)
			},
			expectedMsg: nil,
		},
		"first call to store Get returns nil": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, nil, v1.Scheduled)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(nil),
					s.store.EXPECT().AddOrUpdateVirtualMachineInstance(
						gomock.Eq(ownerUID),
						gomock.Eq(vmiNamespace),
						gomock.Eq(nilPtr),
						gomock.Eq(false)).Times(1),
				)
			},
			expectedMsg: nil,
		},
		"update state of virtual machine": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachineInstance(vmiUUID, vmiName, vmiNamespace, ownerUID, &vsockVal, v1.Running)),
			expectFn: func() {
				gomock.InOrder(
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  nilPtr,
						Running:   false,
					}),
					s.store.EXPECT().AddOrUpdateVirtualMachineInstance(
						gomock.Eq(ownerUID),
						gomock.Eq(vmiNamespace),
						gomock.Eq(&vsockVal),
						gomock.Eq(true)).Times(1),
					s.store.EXPECT().Get(gomock.Eq(ownerUID)).Times(1).Return(&store.VirtualMachineInfo{
						UID:       ownerUID,
						Name:      vmiName,
						Namespace: vmiNamespace,
						VSOCKCID:  &vsockVal,
						Running:   true,
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

func newVirtualMachineInstance(uid, name, namespace, owner string, vsock *uint32, phase v1.VirtualMachineInstancePhase) *v1.VirtualMachineInstance {
	return &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID(uid),
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					UID: types.UID(owner),
				},
			},
		},
		Status: v1.VirtualMachineInstanceStatus{
			Phase:    phase,
			VSOCKCID: vsock,
		},
	}
}

func toUnstructured(obj any) *unstructured.Unstructured {
	ret := &unstructured.Unstructured{}
	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	tmp, _ := json.Marshal(unstructuredObj)
	_ = ret.UnmarshalJSON(tmp)
	return ret
}
