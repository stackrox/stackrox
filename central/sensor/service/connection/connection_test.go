package connection

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	clusterMgrMock "github.com/stackrox/rox/central/sensor/service/common/mocks"
	testutilsMTLS "github.com/stackrox/rox/central/testutils/mtls"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/testutils/envisolator"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
	envIsolator *envisolator.EnvIsolator
}

func (s *testSuite) SetupSuite() {
	s.envIsolator = envisolator.NewEnvIsolator(s.T())
}

func (s *testSuite) TearDownTest() {
	s.envIsolator.RestoreAll()
}

func (s *testSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.envIsolator)
	s.Require().NoError(err)
}

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
func (s *testSuite) TestGetPolicySyncMsgFromPolicies() {
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
	s.NoError(err)

	policySync := msg.GetPolicySync()
	s.NotNil(policySync)
	s.NotEmpty(policySync.Policies)
	s.Equal(sensorVersion.String(), policySync.Policies[0].GetPolicyVersion())
}

func (s *testSuite) TestSendsAuditLogSyncMessageIfEnabledOnRun() {
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

	ctrl := gomock.NewController(s.T())
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

	s.NoError(sensorMockConn.Run(ctx, server, caps))

	for _, msg := range server.sentList {
		if syncMsg := msg.GetAuditLogSync(); syncMsg != nil {
			s.Equal(auditLogState, syncMsg.GetNodeAuditLogFileStates())
			return
		}
	}

	s.FailNow("Audit log sync message was not sent")
}

func (s *testSuite) TestIssueLocalScannerCerts() {
	s.envIsolator.Setenv(features.LocalImageScanning.EnvVar(), "true")
	if !features.LocalImageScanning.Enabled() {
		s.T().Skip()
	}
	testCases := map[string]struct {
		clusterID  string
		shouldFail bool
	}{
		"empty cluster id":     {"", true},
		"non empty cluster id": {"clusterID", false},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			sendC := make(chan *central.MsgToSensor)
			sensorMockConn := &sensorConnection{
				clusterID: tc.clusterID,
				sendC:     sendC,
				stopSig:   concurrency.NewErrorSignal(),
			}
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			namespace := "namespace"
			request := &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
					IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{Namespace: namespace},
				},
			}

			go func() {
				s.NoError(sensorMockConn.handleMessage(ctx, request))
			}()

			select {
			case msgToSensor := <-sendC:
				if tc.shouldFail {
					s.NotNil(msgToSensor.GetIssueLocalScannerCertsResponse().GetError())
				} else {
					s.NotNil(msgToSensor.GetIssueLocalScannerCertsResponse().GetCertificates())
				}
			case <-ctx.Done():
				s.Fail(ctx.Err().Error())
			}
		})
	}
}
