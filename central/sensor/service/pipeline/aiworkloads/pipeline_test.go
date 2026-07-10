package aiworkloads

import (
	"context"
	"testing"

	aiWorkloadDSMocks "github.com/stackrox/rox/central/aiworkload/datastore/mocks"
	clusterDSMocks "github.com/stackrox/rox/central/cluster/datastore/mocks"
	aiworkloadV1 "github.com/stackrox/rox/generated/internalapi/aiworkload/v1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCapabilities(t *testing.T) {
	pipeline := &pipelineImpl{}
	assert.ElementsMatch(
		t,
		[]centralsensor.CentralCapability{centralsensor.AIWorkloadsSupported},
		pipeline.Capabilities(),
	)
}

func TestMatch(t *testing.T) {
	pipeline := &pipelineImpl{}

	tests := map[string]struct {
		input *central.MsgFromSensor
		want  bool
	}{
		"nil input": {
			input: nil,
			want:  false,
		},
		"empty input": {
			input: &central.MsgFromSensor{},
			want:  false,
		},
		"wrong message type": {
			input: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_Deployment{},
					},
				},
			},
			want: false,
		},
		"correct message type": {
			input: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_AiWorkload{
							AiWorkload: &aiworkloadV1.AIWorkload{Id: "test-id"},
						},
					},
				},
			},
			want: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, pipeline.Match(tc.input))
		})
	}
}

type pipelineMocks struct {
	clusters    *clusterDSMocks.MockDataStore
	aiWorkloads *aiWorkloadDSMocks.MockDataStore
}

func TestRun(t *testing.T) {
	tests := map[string]struct {
		setupMocks   func(*pipelineMocks)
		message      *central.MsgFromSensor
		expectsError bool
	}{
		"nil ai workload returns error": {
			message: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Resource: &central.SensorEvent_Node{
							Node: &storage.Node{Id: "node-1"},
						},
					},
				},
			},
			expectsError: true,
		},
		"create resource upserts to datastore": {
			setupMocks: func(m *pipelineMocks) {
				m.clusters.EXPECT().GetClusterName(gomock.Any(), "cluster-1").Return("my-cluster", true, nil)
				m.aiWorkloads.EXPECT().UpsertAIWorkload(gomock.Any(), gomock.Any()).Return(nil)
			},
			message: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Action: central.ResourceAction_CREATE_RESOURCE,
						Resource: &central.SensorEvent_AiWorkload{
							AiWorkload: &aiworkloadV1.AIWorkload{
								Id:          "test-id",
								Name:        "test-model",
								ClusterId:   "cluster-1",
								ModelFormat: "vLLM",
							},
						},
					},
				},
			},
		},
		"update resource upserts to datastore": {
			setupMocks: func(m *pipelineMocks) {
				m.clusters.EXPECT().GetClusterName(gomock.Any(), "cluster-1").Return("my-cluster", true, nil)
				m.aiWorkloads.EXPECT().UpsertAIWorkload(gomock.Any(), gomock.Any()).Return(nil)
			},
			message: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Action: central.ResourceAction_UPDATE_RESOURCE,
						Resource: &central.SensorEvent_AiWorkload{
							AiWorkload: &aiworkloadV1.AIWorkload{
								Id:        "test-id",
								ClusterId: "cluster-1",
							},
						},
					},
				},
			},
		},
		"remove resource deletes from datastore": {
			setupMocks: func(m *pipelineMocks) {
				m.aiWorkloads.EXPECT().DeleteAIWorkloads(gomock.Any(), "test-id").Return(nil)
			},
			message: &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_Event{
					Event: &central.SensorEvent{
						Action: central.ResourceAction_REMOVE_RESOURCE,
						Resource: &central.SensorEvent_AiWorkload{
							AiWorkload: &aiworkloadV1.AIWorkload{Id: "test-id"},
						},
					},
				},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			m := &pipelineMocks{
				clusters:    clusterDSMocks.NewMockDataStore(ctrl),
				aiWorkloads: aiWorkloadDSMocks.NewMockDataStore(ctrl),
			}
			if tc.setupMocks != nil {
				tc.setupMocks(m)
			}

			p := newPipeline(m.clusters, m.aiWorkloads)
			err := p.Run(context.Background(), "cluster-1", tc.message, nil)
			if tc.expectsError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
