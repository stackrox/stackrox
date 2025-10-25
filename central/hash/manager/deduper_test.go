package manager

import (
	"testing"

	"github.com/stackrox/rox/generated/internalapi/central"
	v4 "github.com/stackrox/rox/generated/internalapi/scanner/v4"
	v1 "github.com/stackrox/rox/generated/internalapi/virtualmachine/v1"
	"github.com/stackrox/rox/generated/storage"
	eventPkg "github.com/stackrox/rox/pkg/sensor/event"
	"github.com/stackrox/rox/pkg/sensor/hash"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/assert"
)

func getDeploymentEvent(action central.ResourceAction, id, name string, processingAttempt int32) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: id,
				Resource: &central.SensorEvent_Deployment{
					Deployment: &storage.Deployment{
						Id:   id,
						Name: name,
					},
				},
				Action: action,
			}},
		ProcessingAttempt: processingAttempt,
	}
}

func TestDeduper(t *testing.T) {
	type testEvents struct {
		event  *central.MsgFromSensor
		result bool
	}
	cases := []struct {
		testName   string
		testEvents []testEvents
	}{
		{
			testName: "empty event",
			testEvents: []testEvents{
				{
					event:  &central.MsgFromSensor{},
					result: true,
				},
			},
		},
		{
			testName: "network flow",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_NetworkFlowUpdate{
							NetworkFlowUpdate: &central.NetworkFlowUpdate{},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "duplicate runtime alerts",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "abc",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										Stage: storage.LifecycleStage_RUNTIME,
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "abc",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										Stage: storage.LifecycleStage_RUNTIME,
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "duplicate node indexes should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "1",
								Resource: &central.SensorEvent_IndexReport{
									IndexReport: &v4.IndexReport{
										HashId:   "a",
										State:    "7",
										Success:  true,
										Err:      "",
										Contents: nil,
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "1",
								Resource: &central.SensorEvent_IndexReport{
									IndexReport: &v4.IndexReport{
										HashId:   "a",
										State:    "7",
										Success:  true,
										Err:      "",
										Contents: nil,
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "attempted alert",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "abc",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										Stage: storage.LifecycleStage_DEPLOY,
										Alerts: []*storage.Alert{
											{
												State: storage.ViolationState_ATTEMPTED,
											},
										},
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "process indicator",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Resource: &central.SensorEvent_ProcessIndicator{
									ProcessIndicator: &storage.ProcessIndicator{},
								},
							}},
					},
					result: true,
				},
			},
		},
		{
			testName: "deployment create",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_CREATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment update",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment sync",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment flow",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_CREATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 0),
					result: false,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
			},
		},
		{
			testName: "deployment processing attempt flow",
			testEvents: []testEvents{
				{
					event:  getDeploymentEvent(central.ResourceAction_CREATE_RESOURCE, "1", "dep1", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 1),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep2", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep1", 2),
					result: false,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep2", 0),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_REMOVE_RESOURCE, "1", "dep1", 1),
					result: true,
				},
				{
					event:  getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "1", "dep2", 1),
					result: false,
				},
			},
		},
		{
			testName: "Pod should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "pod1",
								Resource: &central.SensorEvent_Pod{
									Pod: &storage.Pod{
										Id:        "pod1",
										Name:      "test-pod",
										Namespace: "default",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "pod1",
								Resource: &central.SensorEvent_Pod{
									Pod: &storage.Pod{
										Id:        "pod1",
										Name:      "test-pod",
										Namespace: "default",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "pod1",
								Resource: &central.SensorEvent_Pod{
									Pod: &storage.Pod{
										Id:        "pod1",
										Name:      "test-pod",
										Namespace: "default",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "Namespace should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "ns1",
								Resource: &central.SensorEvent_Namespace{
									Namespace: &storage.NamespaceMetadata{
										Name:      "default",
										Id:        "ns1",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "ns1",
								Resource: &central.SensorEvent_Namespace{
									Namespace: &storage.NamespaceMetadata{
										Name:      "default",
										Id:        "ns1",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "ns1",
								Resource: &central.SensorEvent_Namespace{
									Namespace: &storage.NamespaceMetadata{
										Name:      "default",
										Id:        "ns1",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "NetworkPolicy should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "np1",
								Resource: &central.SensorEvent_NetworkPolicy{
									NetworkPolicy: &storage.NetworkPolicy{
										Id:        "np1",
										Name:      "test-policy",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "np1",
								Resource: &central.SensorEvent_NetworkPolicy{
									NetworkPolicy: &storage.NetworkPolicy{
										Id:        "np1",
										Name:      "test-policy",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "np1",
								Resource: &central.SensorEvent_NetworkPolicy{
									NetworkPolicy: &storage.NetworkPolicy{
										Id:        "np1",
										Name:      "test-policy",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "Secret should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "secret1",
								Resource: &central.SensorEvent_Secret{
									Secret: &storage.Secret{
										Id:        "secret1",
										Name:      "test-secret",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "secret1",
								Resource: &central.SensorEvent_Secret{
									Secret: &storage.Secret{
										Id:        "secret1",
										Name:      "test-secret",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "secret1",
								Resource: &central.SensorEvent_Secret{
									Secret: &storage.Secret{
										Id:        "secret1",
										Name:      "test-secret",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "Node should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "node1",
								Resource: &central.SensorEvent_Node{
									Node: &storage.Node{
										Id:        "node1",
										Name:      "test-node",
										ClusterId: "cluster1",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "node1",
								Resource: &central.SensorEvent_Node{
									Node: &storage.Node{
										Id:        "node1",
										Name:      "test-node",
										ClusterId: "cluster1",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "NodeInventory should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "ni1",
								Resource: &central.SensorEvent_NodeInventory{
									NodeInventory: &storage.NodeInventory{
										NodeId:   "ni1",
										NodeName: "test-node",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "ni1",
								Resource: &central.SensorEvent_NodeInventory{
									NodeInventory: &storage.NodeInventory{
										NodeId:   "ni1",
										NodeName: "test-node",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ServiceAccount should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "sa1",
								Resource: &central.SensorEvent_ServiceAccount{
									ServiceAccount: &storage.ServiceAccount{
										Id:        "sa1",
										Name:      "test-sa",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "sa1",
								Resource: &central.SensorEvent_ServiceAccount{
									ServiceAccount: &storage.ServiceAccount{
										Id:        "sa1",
										Name:      "test-sa",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "sa1",
								Resource: &central.SensorEvent_ServiceAccount{
									ServiceAccount: &storage.ServiceAccount{
										Id:        "sa1",
										Name:      "test-sa",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "Role should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "role1",
								Resource: &central.SensorEvent_Role{
									Role: &storage.K8SRole{
										Id:        "role1",
										Name:      "test-role",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "role1",
								Resource: &central.SensorEvent_Role{
									Role: &storage.K8SRole{
										Id:        "role1",
										Name:      "test-role",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "role1",
								Resource: &central.SensorEvent_Role{
									Role: &storage.K8SRole{
										Id:        "role1",
										Name:      "test-role",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "Binding should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "binding1",
								Resource: &central.SensorEvent_Binding{
									Binding: &storage.K8SRoleBinding{
										Id:        "binding1",
										Name:      "test-binding",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "binding1",
								Resource: &central.SensorEvent_Binding{
									Binding: &storage.K8SRoleBinding{
										Id:        "binding1",
										Name:      "test-binding",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "binding1",
								Resource: &central.SensorEvent_Binding{
									Binding: &storage.K8SRoleBinding{
										Id:        "binding1",
										Name:      "test-binding",
										Namespace: "default",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ReprocessDeployment should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "reprocess1",
								Resource: &central.SensorEvent_ReprocessDeployment{
									ReprocessDeployment: &central.ReprocessDeploymentRisk{
										DeploymentId: "dep1",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "reprocess1",
								Resource: &central.SensorEvent_ReprocessDeployment{
									ReprocessDeployment: &central.ReprocessDeploymentRisk{
										DeploymentId: "dep1",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ProviderMetadata should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "metadata1",
								Resource: &central.SensorEvent_ProviderMetadata{
									ProviderMetadata: &storage.ProviderMetadata{
										Region: "us-east-1",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "metadata1",
								Resource: &central.SensorEvent_ProviderMetadata{
									ProviderMetadata: &storage.ProviderMetadata{
										Region: "us-east-1",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "OrchestratorMetadata should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "orch1",
								Resource: &central.SensorEvent_OrchestratorMetadata{
									OrchestratorMetadata: &storage.OrchestratorMetadata{
										Version: "1.24.0",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "orch1",
								Resource: &central.SensorEvent_OrchestratorMetadata{
									OrchestratorMetadata: &storage.OrchestratorMetadata{
										Version: "1.24.0",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ImageIntegration should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "img1",
								Resource: &central.SensorEvent_ImageIntegration{
									ImageIntegration: &storage.ImageIntegration{
										Id:   "img1",
										Name: "test-integration",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "img1",
								Resource: &central.SensorEvent_ImageIntegration{
									ImageIntegration: &storage.ImageIntegration{
										Id:   "img1",
										Name: "test-integration",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "VirtualMachine should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "vm1",
								Resource: &central.SensorEvent_VirtualMachine{
									VirtualMachine: &v1.VirtualMachine{
										Id:        "vm1",
										Name:      "test-vm",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "vm1",
								Resource: &central.SensorEvent_VirtualMachine{
									VirtualMachine: &v1.VirtualMachine{
										Id:        "vm1",
										Name:      "test-vm",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "vm1",
								Resource: &central.SensorEvent_VirtualMachine{
									VirtualMachine: &v1.VirtualMachine{
										Id:        "vm1",
										Name:      "test-vm",
										ClusterId: "cluster1",
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "VirtualMachineIndexReport should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "vmindex1",
								Resource: &central.SensorEvent_VirtualMachineIndexReport{
									VirtualMachineIndexReport: &v1.IndexReportEvent{
										Id: "vm1",
										Index: &v1.IndexReport{
											VsockCid: "1",
											IndexV4: &v4.IndexReport{
												HashId:   "vmhash1",
												State:    "7",
												Success:  true,
												Contents: nil,
											},
										},
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "vmindex1",
								Resource: &central.SensorEvent_VirtualMachineIndexReport{
									VirtualMachineIndexReport: &v1.IndexReportEvent{
										Id: "vm1",
										Index: &v1.IndexReport{
											VsockCid: "1",
											IndexV4: &v4.IndexReport{
												HashId:   "vmhash1",
												State:    "7",
												Success:  true,
												Contents: nil,
											},
										},
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "Deploy AlertResults (not runtime, not resolved, not attempted) should be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "alert1",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										DeploymentId: "dep1",
										Stage:        storage.LifecycleStage_DEPLOY,
										Alerts: []*storage.Alert{
											{
												State: storage.ViolationState_ACTIVE,
											},
										},
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "alert1",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										DeploymentId: "dep1",
										Stage:        storage.LifecycleStage_DEPLOY,
										Alerts: []*storage.Alert{
											{
												State: storage.ViolationState_ACTIVE,
											},
										},
									},
								},
								Action: central.ResourceAction_CREATE_RESOURCE,
							},
						},
					},
					result: false,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "alert1",
								Resource: &central.SensorEvent_AlertResults{
									AlertResults: &central.AlertResults{
										DeploymentId: "dep1",
										Stage:        storage.LifecycleStage_DEPLOY,
										Alerts: []*storage.Alert{
											{
												State: storage.ViolationState_ACTIVE,
											},
										},
									},
								},
								Action: central.ResourceAction_REMOVE_RESOURCE,
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorResult should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cor1",
								Resource: &central.SensorEvent_ComplianceOperatorResult{
									ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{
										CheckId:   "check1",
										CheckName: "test-check",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cor1",
								Resource: &central.SensorEvent_ComplianceOperatorResult{
									ComplianceOperatorResult: &storage.ComplianceOperatorCheckResult{
										CheckId:   "check1",
										CheckName: "test-check",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorProfile should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cop1",
								Resource: &central.SensorEvent_ComplianceOperatorProfile{
									ComplianceOperatorProfile: &storage.ComplianceOperatorProfile{
										Id:   "cop1",
										Name: "test-profile",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cop1",
								Resource: &central.SensorEvent_ComplianceOperatorProfile{
									ComplianceOperatorProfile: &storage.ComplianceOperatorProfile{
										Id:   "cop1",
										Name: "test-profile",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorRule should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cor1",
								Resource: &central.SensorEvent_ComplianceOperatorRule{
									ComplianceOperatorRule: &storage.ComplianceOperatorRule{
										Id:   "cor1",
										Name: "test-rule",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cor1",
								Resource: &central.SensorEvent_ComplianceOperatorRule{
									ComplianceOperatorRule: &storage.ComplianceOperatorRule{
										Id:   "cor1",
										Name: "test-rule",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorScanSettingBinding should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cossb1",
								Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{
									ComplianceOperatorScanSettingBinding: &storage.ComplianceOperatorScanSettingBinding{
										Id:   "cossb1",
										Name: "test-binding",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cossb1",
								Resource: &central.SensorEvent_ComplianceOperatorScanSettingBinding{
									ComplianceOperatorScanSettingBinding: &storage.ComplianceOperatorScanSettingBinding{
										Id:   "cossb1",
										Name: "test-binding",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorScan should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cos1",
								Resource: &central.SensorEvent_ComplianceOperatorScan{
									ComplianceOperatorScan: &storage.ComplianceOperatorScan{
										Id:   "cos1",
										Name: "test-scan",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cos1",
								Resource: &central.SensorEvent_ComplianceOperatorScan{
									ComplianceOperatorScan: &storage.ComplianceOperatorScan{
										Id:   "cos1",
										Name: "test-scan",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorResultV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "corv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorResultV2{
									ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{
										CheckId:   "checkv2_1",
										CheckName: "test-check-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "corv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorResultV2{
									ComplianceOperatorResultV2: &central.ComplianceOperatorCheckResultV2{
										CheckId:   "checkv2_1",
										CheckName: "test-check-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorProfileV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "copv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
									ComplianceOperatorProfileV2: &central.ComplianceOperatorProfileV2{
										Id:   "copv2_1",
										Name: "test-profile-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "copv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorProfileV2{
									ComplianceOperatorProfileV2: &central.ComplianceOperatorProfileV2{
										Id:   "copv2_1",
										Name: "test-profile-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorRuleV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "corulev2_1",
								Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
									ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
										Id:   "corulev2_1",
										Name: "test-rule-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "corulev2_1",
								Resource: &central.SensorEvent_ComplianceOperatorRuleV2{
									ComplianceOperatorRuleV2: &central.ComplianceOperatorRuleV2{
										Id:   "corulev2_1",
										Name: "test-rule-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorScanV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cosv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorScanV2{
									ComplianceOperatorScanV2: &central.ComplianceOperatorScanV2{
										Id:   "cosv2_1",
										Name: "test-scan-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cosv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorScanV2{
									ComplianceOperatorScanV2: &central.ComplianceOperatorScanV2{
										Id:   "cosv2_1",
										Name: "test-scan-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorScanSettingBindingV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cossbv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
									ComplianceOperatorScanSettingBindingV2: &central.ComplianceOperatorScanSettingBindingV2{
										Id:   "cossbv2_1",
										Name: "test-binding-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cossbv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorScanSettingBindingV2{
									ComplianceOperatorScanSettingBindingV2: &central.ComplianceOperatorScanSettingBindingV2{
										Id:   "cossbv2_1",
										Name: "test-binding-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorSuiteV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cosuitv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorSuiteV2{
									ComplianceOperatorSuiteV2: &central.ComplianceOperatorSuiteV2{
										Id:   "cosuitv2_1",
										Name: "test-suite-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cosuitv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorSuiteV2{
									ComplianceOperatorSuiteV2: &central.ComplianceOperatorSuiteV2{
										Id:   "cosuitv2_1",
										Name: "test-suite-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
		{
			testName: "ComplianceOperatorRemediationV2 should not be deduped",
			testEvents: []testEvents{
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cormedv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorRemediationV2{
									ComplianceOperatorRemediationV2: &central.ComplianceOperatorRemediationV2{
										Id:   "cormedv2_1",
										Name: "test-remediation-v2",
									},
								},
							},
						},
					},
					result: true,
				},
				{
					event: &central.MsgFromSensor{
						Msg: &central.MsgFromSensor_Event{
							Event: &central.SensorEvent{
								Id: "cormedv2_1",
								Resource: &central.SensorEvent_ComplianceOperatorRemediationV2{
									ComplianceOperatorRemediationV2: &central.ComplianceOperatorRemediationV2{
										Id:   "cormedv2_1",
										Name: "test-remediation-v2",
									},
								},
							},
						},
					},
					result: true,
				},
			},
		},
	}
	for _, c := range cases {
		testCase := c
		t.Run(c.testName, func(t *testing.T) {
			deduper := NewDeduper(make(map[string]uint64), uuid.NewV4().String()).(*deduperImpl)
			for _, testEvent := range testCase.testEvents {
				assert.Equal(t, testEvent.result, deduper.ShouldProcess(testEvent.event))
			}
			assert.Len(t, deduper.successfullyProcessed, 0)
			assert.Len(t, deduper.received, 0)
		})
	}
}

func TestReconciliation(t *testing.T) {
	deduper := NewDeduper(make(map[string]uint64), uuid.NewV4().String()).(*deduperImpl)

	d1 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "1", "1", 0)
	d2 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "2", "2", 0)
	d3 := getDeploymentEvent(central.ResourceAction_UPDATE_RESOURCE, "3", "3", 0)
	d4 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "4", "4", 0)
	d5 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "5", "5", 0)

	d1Alert := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{
				Id: d1.GetEvent().GetId(),
				Resource: &central.SensorEvent_AlertResults{
					AlertResults: &central.AlertResults{
						DeploymentId: d1.GetEvent().GetId(),
						Stage:        storage.LifecycleStage_DEPLOY,
					},
				},
				Action: central.ResourceAction_SYNC_RESOURCE,
			},
		},
	}

	// Basic case
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.MarkSuccessful(d1)
	deduper.ShouldProcess(d2)
	deduper.MarkSuccessful(d2)
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 2)
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d1))
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d2))

	// Values in successfully processed that should be removed
	deduper.ShouldProcess(d3)
	deduper.MarkSuccessful(d3)

	deduper.StartSync()
	deduper.ShouldProcess(d4)
	deduper.ShouldProcess(d5)
	deduper.MarkSuccessful(d4)
	deduper.MarkSuccessful(d5)
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 2)
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d4))
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d5))

	// Should clear out successfully processed
	deduper.StartSync()
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 0)

	// Add d1 to successfully processed map, call start sync again, and only put d1 in the received map
	// and not in successfully processed. Ensure it is not reconciled away
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.MarkSuccessful(d1)
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 1)
	assert.Contains(t, deduper.successfullyProcessed, eventPkg.GetKeyFromMessage(d1))

	deduper.StartSync()
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 0)

	// Ensure alert is removed when reconcile occurs
	deduper.StartSync()
	deduper.ShouldProcess(d1)
	deduper.MarkSuccessful(d1)
	deduper.ShouldProcess(d1Alert)
	deduper.MarkSuccessful(d1Alert)
	assert.Len(t, deduper.successfullyProcessed, 2)
	deduper.StartSync()
	deduper.ProcessSync()
	assert.Len(t, deduper.successfullyProcessed, 0)
}

type testEvents func(*testing.T, **deduperImpl)

func TestReconciliationOnDisconnection(t *testing.T) {
	d1 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "1", "d1", 0)
	d2 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "2", "d2", 0)
	d3 := getDeploymentEvent(central.ResourceAction_SYNC_RESOURCE, "3", "d3", 0)
	cases := map[string]struct {
		events []testEvents
	}{
		"normal sync": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"normal sync with initial deduper state (sensor cannot handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(getHashesFromEvents([]*central.MsgFromSensor{d1, d2})),
				// Sync resources d1 d2 as sensor cannot handle the deduper state
				syncEventShouldNotProcess(d1),
				syncEventShouldNotProcess(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"normal sync with initial deduper state (sensor can handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(getHashesFromEvents([]*central.MsgFromSensor{d1, d2})),
				// Sensor does not send the sync resources d1 d2 as it can handle the deduper state
				syncEventSuccessfully(d3),
				// Sync event
				syncEvent,
				// Assert d1, d2, and d3
				assertEvents([]*central.MsgFromSensor{ /* d1, d2, */ d3}),
			},
		},
		"reconnection (sensor cannot handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
				// Simulated reconnection
				newConnection(nil),
				// Sensor sends the sync resources d1 d2 as it cannot handle the deduper state
				// Both event should not be processed as they are already in the successfullyProcessed map
				syncEventShouldNotProcess(d1),
				syncEventShouldNotProcess(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"reconnection with unsuccessful events (sensor cannot handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventUnsuccessfully(d2),
				// Simulated reconnection
				newConnection(nil),
				// Sensor sends the sync resources d1 d2 again as it cannot handle the deduper state
				// d1 should not be processed as it is already in the successfully processed map
				syncEventShouldNotProcess(d1),
				// d2 should be processed
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
			},
		},
		"reconnection (sensor can handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventSuccessfully(d2),
				// Sync event
				syncEvent,
				// Assert d1 and d2
				assertEvents([]*central.MsgFromSensor{d1, d2}),
				// Simulated reconnection
				newConnection(nil),
				// Sensor does not send the sync resources d1 d2 as it can handle the deduper state
				syncEventSuccessfully(d3),
				// Sync event
				syncEvent,
				// Assert d1, d2, and d3
				assertEvents([]*central.MsgFromSensor{ /* d1, d2, */ d3}), // FIXME: we should have d1 and d2
			},
		},
		"reconnection with unsuccessful events (sensor can handle the deduper state)": {
			events: []testEvents{
				// Simulated new connection
				newConnection(nil),
				// Sync resources d1 d2
				syncEventSuccessfully(d1),
				syncEventUnsuccessfully(d2),
				// Simulated reconnection
				newConnection(nil),
				// Sensor does not send the sync resources d1 as it can handle the deduper state
				syncEventSuccessfully(d2),
				syncEventSuccessfully(d3),
				// Sync event
				syncEvent,
				// Assert d1, d2, and d3
				assertEvents([]*central.MsgFromSensor{ /* d1, */ d2, d3}),
			},
		},
	}
	for tname, tc := range cases {
		t.Run(tname, func(tt *testing.T) {
			var deduper *deduperImpl
			for _, event := range tc.events {
				event(tt, &deduper)
			}
		})
	}
}

func newConnection(initialHashes map[string]uint64) testEvents {
	return func(_ *testing.T, deduper **deduperImpl) {
		if *deduper == nil {
			*deduper = NewDeduper(initialHashes, uuid.NewV4().String()).(*deduperImpl)
		}
		(*deduper).StartSync()
	}
}

func syncEventSuccessfully(event *central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.True(t, (*deduper).ShouldProcess(event))
		(*deduper).MarkSuccessful(event)
	}
}

func syncEventShouldNotProcess(event *central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.False(t, (*deduper).ShouldProcess(event))
	}
}

func syncEventUnsuccessfully(event *central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.True(t, (*deduper).ShouldProcess(event))
	}
}

func syncEvent(_ *testing.T, deduper **deduperImpl) {
	(*deduper).ProcessSync()
}

func assertEvents(events []*central.MsgFromSensor) testEvents {
	return func(t *testing.T, deduper **deduperImpl) {
		assert.Len(t, (*deduper).successfullyProcessed, len(events))
		for _, event := range events {
			_, ok := (*deduper).successfullyProcessed[eventPkg.GetKeyFromMessage(event)]
			assert.True(t, ok)
		}
	}
}

func getHashesFromEvents(events []*central.MsgFromSensor) map[string]uint64 {
	ret := make(map[string]uint64)
	hasher := hash.NewHasher()
	for _, event := range events {
		hashValue, _ := hasher.HashEvent(event.GetEvent())
		ret[eventPkg.GetKeyFromMessage(event)] = hashValue
	}
	return ret
}
