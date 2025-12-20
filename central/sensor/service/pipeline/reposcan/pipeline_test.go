package reposcan

import (
	"context"
	"testing"

	"github.com/stackrox/rox/central/sensor/service/common"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stretchr/testify/suite"
)

func TestPipeline(t *testing.T) {
	suite.Run(t, &PipelineTestSuite{})
}

type PipelineTestSuite struct {
	suite.Suite
}

// mockRepoScanBroker implements RepoScanBroker for testing.
type mockRepoScanBroker struct {
	receivedResponses    []*central.RepoScanResponse
	disconnectedClusters []string
}

func (m *mockRepoScanBroker) OnScanResponse(clusterID string, msg *central.RepoScanResponse) {
	m.receivedResponses = append(m.receivedResponses, msg)
}

func (m *mockRepoScanBroker) OnClusterDisconnect(clusterID string) {
	m.disconnectedClusters = append(m.disconnectedClusters, clusterID)
}

// TestMatchRepoScanResponse verifies the pipeline matches RepoScanResponse messages.
func (s *PipelineTestSuite) TestMatchRepoScanResponse() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	// Should match RepoScanResponse.
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "req-1",
				Payload: &central.RepoScanResponse_Start_{
					Start: &central.RepoScanResponse_Start{},
				},
			},
		},
	}
	s.True(pipeline.Match(msg))

	// Should not match other message types.
	msg = &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: &central.SensorEvent{},
		},
	}
	s.False(pipeline.Match(msg))
}

// TestRunForwardsToRoker verifies Run forwards messages to the broker.
func (s *PipelineTestSuite) TestRunForwardsToRoker() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	resp := &central.RepoScanResponse{
		RequestId: "req-1",
		Payload: &central.RepoScanResponse_Start_{
			Start: &central.RepoScanResponse_Start{},
		},
	}

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: resp,
		},
	}

	// mockMessageInjector is not needed for this pipeline.
	err := pipeline.Run(context.Background(), "cluster-1", msg, nil)
	s.NoError(err)

	// Verify broker received the message.
	s.Len(broker.receivedResponses, 1)
	s.Equal(resp, broker.receivedResponses[0])
}

// TestRunWithMultipleMessages verifies multiple messages are forwarded correctly.
func (s *PipelineTestSuite) TestRunWithMultipleMessages() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	// Send Start.
	msg1 := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "req-1",
				Payload: &central.RepoScanResponse_Start_{
					Start: &central.RepoScanResponse_Start{},
				},
			},
		},
	}

	err := pipeline.Run(context.Background(), "cluster-1", msg1, nil)
	s.NoError(err)

	// Send Update.
	msg2 := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "req-1",
				Payload: &central.RepoScanResponse_Update_{
					Update: &central.RepoScanResponse_Update{
						Tag: "latest",
						Outcome: &central.RepoScanResponse_Update_Metadata{
							Metadata: &central.TagMetadata{
								ManifestDigest: "sha256:abc123",
							},
						},
					},
				},
			},
		},
	}

	err = pipeline.Run(context.Background(), "cluster-1", msg2, nil)
	s.NoError(err)

	// Send End.
	msg3 := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "req-1",
				Payload: &central.RepoScanResponse_End_{
					End: &central.RepoScanResponse_End{
						Success:         true,
						SuccessfulCount: 1,
					},
				},
			},
		},
	}

	err = pipeline.Run(context.Background(), "cluster-1", msg3, nil)
	s.NoError(err)

	// Verify all messages were forwarded.
	s.Len(broker.receivedResponses, 3)
	s.Equal("req-1", broker.receivedResponses[0].GetRequestId())
	s.NotNil(broker.receivedResponses[0].GetStart())
	s.Equal("req-1", broker.receivedResponses[1].GetRequestId())
	s.NotNil(broker.receivedResponses[1].GetUpdate())
	s.Equal("req-1", broker.receivedResponses[2].GetRequestId())
	s.NotNil(broker.receivedResponses[2].GetEnd())
}

// TestOnFinishCleansUpCluster verifies OnFinish calls broker cleanup.
func (s *PipelineTestSuite) TestOnFinishCleansUpCluster() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	pipeline.OnFinish("cluster-1")

	// Verify broker was notified of disconnect.
	s.Len(broker.disconnectedClusters, 1)
	s.Equal("cluster-1", broker.disconnectedClusters[0])
}

// TestOnFinishWithMultipleClusters verifies each cluster is cleaned up independently.
func (s *PipelineTestSuite) TestOnFinishWithMultipleClusters() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	pipeline.OnFinish("cluster-1")
	pipeline.OnFinish("cluster-2")
	pipeline.OnFinish("cluster-3")

	// Verify all clusters were cleaned up.
	s.Len(broker.disconnectedClusters, 3)
	s.Contains(broker.disconnectedClusters, "cluster-1")
	s.Contains(broker.disconnectedClusters, "cluster-2")
	s.Contains(broker.disconnectedClusters, "cluster-3")
}

// TestCapabilities verifies the pipeline returns nil capabilities.
func (s *PipelineTestSuite) TestCapabilities() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	caps := pipeline.Capabilities()
	s.Nil(caps)
}

// TestReconcile verifies Reconcile is a no-op.
func (s *PipelineTestSuite) TestReconcile() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	err := pipeline.Reconcile(context.Background(), "cluster-1", &reconciliation.StoreMap{})
	s.NoError(err)
}

// TestRunWithNilMessage verifies error handling for nil RepoScanResponse.
func (s *PipelineTestSuite) TestRunWithNilMessage() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: nil,
		},
	}

	err := pipeline.Run(context.Background(), "cluster-1", msg, nil)
	s.Error(err)
	s.Contains(err.Error(), "reposcan request is nil")

	// Broker should NOT receive anything since validation failed.
	s.Len(broker.receivedResponses, 0)
}

// TestGetPipelineUsesSingleton verifies GetPipeline uses the broker singleton.
func (s *PipelineTestSuite) TestGetPipelineUsesSingleton() {
	// This test just verifies GetPipeline returns a pipeline without panicking.
	// We can't easily test that it uses the singleton without mocking the singleton.
	pipeline := GetPipeline()
	s.NotNil(pipeline)

	// Verify it implements the interface.
	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{},
		},
	}
	s.True(pipeline.Match(msg))
}

// TestRunPreservesContext verifies context is passed through (even though unused).
func (s *PipelineTestSuite) TestRunPreservesContext() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "req-1",
			},
		},
	}

	// Should complete even with a cancellable context.
	err := pipeline.Run(ctx, "cluster-1", msg, nil)
	s.NoError(err)

	s.Len(broker.receivedResponses, 1)
}

// TestRunWithEmptyRequestID verifies error handling for empty request ID.
func (s *PipelineTestSuite) TestRunWithEmptyRequestID() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "", // Empty request ID
				Payload: &central.RepoScanResponse_Start_{
					Start: &central.RepoScanResponse_Start{},
				},
			},
		},
	}

	err := pipeline.Run(context.Background(), "cluster-1", msg, nil)
	s.Error(err)
	s.Contains(err.Error(), "reposcan request id is empty")

	// Broker should NOT receive anything since validation failed.
	s.Len(broker.receivedResponses, 0)
}

// TestMatchReturnsFalseForNilRepoScanResponse verifies Match returns false when message is nil.
func (s *PipelineTestSuite) TestMatchReturnsFalseForNilRepoScanResponse() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: nil,
		},
	}

	// GetRepoScanResponse() returns nil, so Match should return false.
	s.False(pipeline.Match(msg))
}

// TestRunWithMessageInjector verifies Run ignores the message injector parameter.
func (s *PipelineTestSuite) TestRunWithMessageInjector() {
	broker := &mockRepoScanBroker{}
	pipeline := NewPipeline(broker)

	// mockMessageInjector for testing.
	type mockMessageInjector struct {
		common.MessageInjector
	}

	msg := &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_RepoScanResponse{
			RepoScanResponse: &central.RepoScanResponse{
				RequestId: "req-1",
			},
		},
	}

	// Should work with non-nil injector (even though it's not used).
	err := pipeline.Run(context.Background(), "cluster-1", msg, &mockMessageInjector{})
	s.NoError(err)

	s.Len(broker.receivedResponses, 1)
}
