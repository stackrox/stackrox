package dispatcher

import (
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
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
)

const (
	vmUUID      = "vm-id"
	vmName      = "vm-name"
	vmNamespace = "vm-namespace"
)

func TestVirtualMachinesDispatcher(t *testing.T) {
	suite.Run(t, new(virtualMachineSuite))
}

type virtualMachineSuite struct {
	suite.Suite
	mockCtrl   *gomock.Controller
	store      *mocks.MockvirtualMachineStore
	dispatcher *VirtualMachineDispatcher
}

var _ suite.SetupSubTest = (*virtualMachineSuite)(nil)
var _ suite.TearDownSubTest = (*virtualMachineSuite)(nil)

func (s *virtualMachineSuite) SetupSubTest() {
	s.mockCtrl = gomock.NewController(s.T())
	s.store = mocks.NewMockvirtualMachineStore(s.mockCtrl)
	s.dispatcher = NewVirtualMachineDispatcher(clusterID, s.store)
}

func (s *virtualMachineSuite) TearDownSubTest() {
	s.mockCtrl.Finish()
}

func (s *virtualMachineSuite) Test_VirtualMachineEvents() {
	cases := map[string]struct {
		action      central.ResourceAction
		obj         any
		expectFn    func()
		expectedMsg *component.ResourceEvent
	}{
		"sync event": {
			action: central.ResourceAction_SYNC_RESOURCE,
			obj:    toUnstructured(newVirtualMachine(vmUUID, vmName, vmNamespace, v1.VirtualMachineStatusStopped)),
			expectFn: func() {
				s.store.EXPECT().AddOrUpdate(
					gomock.Eq(&store.VirtualMachineInfo{
						UID:       vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						Running:   false,
					})).Times(1)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmUUID,
				Action: central.ResourceAction_SYNC_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						ClusterId: clusterID,
					},
				},
			}),
		},
		"create event": {
			action: central.ResourceAction_CREATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachine(vmUUID, vmName, vmNamespace, v1.VirtualMachineStatusStopped)),
			expectFn: func() {
				s.store.EXPECT().AddOrUpdate(
					gomock.Eq(&store.VirtualMachineInfo{
						UID:       vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						Running:   false,
					})).Times(1)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmUUID,
				Action: central.ResourceAction_CREATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						ClusterId: clusterID,
					},
				},
			}),
		},
		"update event": {
			action: central.ResourceAction_UPDATE_RESOURCE,
			obj:    toUnstructured(newVirtualMachine(vmUUID, vmName, vmNamespace, v1.VirtualMachineStatusStopped)),
			expectFn: func() {
				s.store.EXPECT().AddOrUpdate(
					gomock.Eq(&store.VirtualMachineInfo{
						UID:       vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						Running:   false,
					})).Times(1)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmUUID,
				Action: central.ResourceAction_UPDATE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						ClusterId: clusterID,
					},
				},
			}),
		},
		"remove event": {
			action: central.ResourceAction_REMOVE_RESOURCE,
			obj:    toUnstructured(newVirtualMachine(vmUUID, vmName, vmNamespace, v1.VirtualMachineStatusStopped)),
			expectFn: func() {
				s.store.EXPECT().Remove(
					gomock.Eq(vmUUID)).Times(1)
			},
			expectedMsg: component.NewEvent(&central.SensorEvent{
				Id:     vmUUID,
				Action: central.ResourceAction_REMOVE_RESOURCE,
				Resource: &central.SensorEvent_VirtualMachine{
					VirtualMachine: &virtualMachineV1.VirtualMachine{
						Id:        vmUUID,
						Name:      vmName,
						Namespace: vmNamespace,
						ClusterId: clusterID,
					},
				},
			}),
		},
		"no unstructured object": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         newVirtualMachine(vmUUID, vmName, vmNamespace, v1.VirtualMachineStatusStopped),
			expectFn:    func() {},
			expectedMsg: nil,
		},
		"no virtual machine": {
			action:      central.ResourceAction_REMOVE_RESOURCE,
			obj:         toUnstructured(&v1.VirtualMachineInstance{}),
			expectFn:    func() {},
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

func newVirtualMachine(uid, name, namespace string, status v1.VirtualMachinePrintableStatus) *v1.VirtualMachine {
	return &v1.VirtualMachine{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID(uid),
			Name:      name,
			Namespace: namespace,
		},
		Status: v1.VirtualMachineStatus{
			PrintableStatus: status,
		},
	}
}
