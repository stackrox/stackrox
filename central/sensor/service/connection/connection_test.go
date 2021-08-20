package connection

import (
	"context"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	clusterMgrMock "github.com/stackrox/rox/central/sensor/service/common/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

type mockServer struct {
	grpc.ServerStream
	sentList []*central.MsgToSensor
}

func (c *mockServer) Send(msg *central.MsgToSensor) error {
	c.sentList = append(c.sentList, msg)
	return nil
}

func (c *mockServer) Recv() (*central.MsgFromSensor, error) {
	return nil, nil
}

// TestGetPolicySyncMsgFromPolicies verifies that the sensor connection is
// capable of downgrading policies to the version known of the underlying
// sensor. The test uses specific policy versions and not a general approach.
func TestGetPolicySyncMsgFromPolicies(t *testing.T) {
	centralVersion := policyversion.CurrentVersion()
	sensorVersion := policyversion.Version1()
	sensorHello := &central.SensorHello{
		PolicyVersion: sensorVersion.String(),
	}
	sensorMockConn := &sensorConnection{
		sensorHello: sensorHello,
	}
	policy := &storage.Policy{
		PolicyVersion: centralVersion.String(),
	}

	msg, err := sensorMockConn.getPolicySyncMsgFromPolicies([]*storage.Policy{policy})
	assert.NoError(t, err)

	policySync := msg.GetPolicySync()
	assert.NotNil(t, policySync)
	assert.NotEmpty(t, policySync.Policies)
	assert.Equal(t, sensorVersion.String(), policySync.Policies[0].GetPolicyVersion())
}

func TestSendsAuditLogSyncMessageIfEnabledOnRun(t *testing.T) {
	envIsolator := envisolator.NewEnvIsolator(t)
	defer envIsolator.RestoreAll()

	ctx := context.Background()
	clusterID := "this-cluster"
	auditLogState := map[string]*storage.AuditLogFileState{
		"node-a": {
			CollectLogsSince: types.TimestampNow(),
			LastAuditId:      "abcd",
		},
	}
	cluster := &storage.Cluster{
		Id:            clusterID,
		DynamicConfig: &storage.DynamicClusterConfig{},
		AuditLogState: auditLogState,
	}

	ctrl := gomock.NewController(t)
	mgrMock := clusterMgrMock.NewMockClusterManager(ctrl)

	sensorMockConn := &sensorConnection{
		clusterID:  clusterID,
		clusterMgr: mgrMock,
	}
	server := &mockServer{
		sentList: make([]*central.MsgToSensor, 0),
	}
	caps := centralsensor.NewSensorCapabilitySet(centralsensor.AuditLogEventsCap)

	mgrMock.EXPECT().GetCluster(ctx, clusterID).Return(cluster, true, nil).AnyTimes()

	assert.NoError(t, sensorMockConn.Run(ctx, server, caps))

	for _, msg := range server.sentList {
		if syncMsg := msg.GetAuditLogSync(); syncMsg != nil {
			assert.Equal(t, auditLogState, syncMsg.GetNodeAuditLogFileStates())
			return
		}
	}

	assert.FailNow(t, "Audit log sync message was not sent")

}
