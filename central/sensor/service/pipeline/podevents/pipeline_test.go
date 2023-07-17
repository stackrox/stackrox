package podevents

import (
	"context"
	"testing"

	podMocks "github.com/stackrox/rox/central/pod/datastore/mocks"
	"github.com/stackrox/rox/central/sensor/service/pipeline"
	"github.com/stackrox/rox/central/sensor/service/pipeline/reconciliation"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/search"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	clusterID = "cluster"
)

var (
	ctx = context.Background()
)

func TestPipeline(t *testing.T) {
	suite.Run(t, new(PipelineTestSuite))
}

type PipelineTestSuite struct {
	suite.Suite

	pods     *podMocks.MockDataStore
	pipeline pipeline.Fragment

	mockCtrl *gomock.Controller
}

func (suite *PipelineTestSuite) SetupTest() {
	suite.mockCtrl = gomock.NewController(suite.T())

	suite.pods = podMocks.NewMockDataStore(suite.mockCtrl)
	suite.pipeline = NewPipeline(suite.pods)
}

func (suite *PipelineTestSuite) TearDownTest() {
	suite.mockCtrl.Finish()
}

func newSensorEvent(_ bool, action central.ResourceAction) *central.SensorEvent {
	return &central.SensorEvent{
		Resource: &central.SensorEvent_Pod{
			Pod: &storage.Pod{
				Id: "id1",
			},
		},
		Action: action,
	}
}

func newMsgFromSensor(event *central.SensorEvent) *central.MsgFromSensor {
	return &central.MsgFromSensor{
		Msg: &central.MsgFromSensor_Event{
			Event: event,
		},
	}
}

func (suite *PipelineTestSuite) TestAddPod() {
	event := newSensorEvent(true, central.ResourceAction_CREATE_RESOURCE)

	expectedPod := event.GetPod()
	suite.pods.EXPECT().UpsertPod(ctx, expectedPod).Return(nil)

	suite.NoError(suite.pipeline.Run(ctx, clusterID, newMsgFromSensor(event), nil))

	suite.Equal(expectedPod, event.GetPod())
}

func (suite *PipelineTestSuite) TestUpdatePod() {
	event := newSensorEvent(true, central.ResourceAction_UPDATE_RESOURCE)

	expectedPod := event.GetPod()
	suite.pods.EXPECT().UpsertPod(ctx, expectedPod).Return(nil)

	suite.NoError(suite.pipeline.Run(ctx, clusterID, newMsgFromSensor(event), nil))

	suite.Equal(expectedPod, event.GetPod())
}

func (suite *PipelineTestSuite) TestRemovePod() {
	event := newSensorEvent(false, central.ResourceAction_REMOVE_RESOURCE)

	expectedPod := event.GetPod()
	suite.pods.EXPECT().RemovePod(ctx, expectedPod.GetId()).Return(nil)

	suite.NoError(suite.pipeline.Run(ctx, clusterID, newMsgFromSensor(event), nil))

	suite.Equal(expectedPod, event.GetPod())
}

func (suite *PipelineTestSuite) TestReconcileNoOp() {
	expectedQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	suite.pods.EXPECT().Search(ctx, expectedQuery).Return(nil, nil)
	suite.NoError(suite.pipeline.Reconcile(ctx, clusterID, reconciliation.NewStoreMap()))
}

func (suite *PipelineTestSuite) TestReconcile() {
	expectedQuery := search.NewQueryBuilder().AddExactMatches(search.ClusterID, clusterID).ProtoQuery()
	expectedPod := fixtures.GetPod()
	result := search.NewResult()
	result.ID = expectedPod.Id
	suite.pods.EXPECT().Search(ctx, expectedQuery).Return([]search.Result{*result}, nil)
	suite.pods.EXPECT().RemovePod(ctx, expectedPod.GetId()).Return(nil)
	suite.NoError(suite.pipeline.Reconcile(ctx, clusterID, reconciliation.NewStoreMap()))
}
