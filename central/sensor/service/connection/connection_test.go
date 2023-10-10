package connection

import (
	"context"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/hash/manager/mocks"
	clusterMgrMock "github.com/stackrox/rox/central/sensor/service/common/mocks"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/set"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
)

func TestHandler(t *testing.T) {
	suite.Run(t, new(testSuite))
}

type testSuite struct {
	suite.Suite
}

func (s *testSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)
}

type mockServer struct {
	grpc.ServerStream
	sentList []*central.MsgToSensor

	errOnDeduperState error
}

func (c *mockServer) Send(msg *central.MsgToSensor) error {
	c.sentList = append(c.sentList, msg)
	if msg.GetDeduperState() != nil {
		return c.errOnDeduperState
	}
	return nil
}

func (c *mockServer) Recv() (*central.MsgFromSensor, error) {
	return nil, nil
}

func (s *testSuite) TestGetPolicySyncMsgFromPoliciesDoesntDowngradeBelowMinimumVersion() {
	sensorMockConn := &sensorConnection{
		sensorHello: &central.SensorHello{
			PolicyVersion: "1",
		},
	}

	msg, err := sensorMockConn.getPolicySyncMsgFromPolicies([]*storage.Policy{{PolicyVersion: policyversion.CurrentVersion().String()}})
	s.NoError(err)

	policySync := msg.GetPolicySync()
	s.Require().NotNil(policySync)
	s.NotEmpty(policySync.Policies)
	s.Equal(policyversion.CurrentVersion().String(), policySync.Policies[0].GetPolicyVersion())
}

func (s *testSuite) TestSendDeduperStateIfSensorReconciliation() {

	cases := map[string]struct {
		givenSensorCapabilities       []centralsensor.SensorCapability
		givenSensorState              central.SensorHello_SensorState
		givenSendError                error
		expectError                   bool
		expectReconciliationMapClosed bool
		expectDeduperStateSent        bool
		expectDeduperStateContents    map[string]uint64
	}{
		"Sensor reconciles: sensor has capability and status is reconnect": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{centralsensor.SensorReconciliationOnReconnect},
			givenSensorState:              central.SensorHello_RECONNECT,
			expectReconciliationMapClosed: true,
			expectDeduperStateSent:        true,
			expectDeduperStateContents:    map[string]uint64{"deployment:1": 0},
		},
		"Sensor reconciles: sensor has capability and status is startup": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{centralsensor.SensorReconciliationOnReconnect},
			givenSensorState:              central.SensorHello_STARTUP,
			expectReconciliationMapClosed: true,
			expectDeduperStateSent:        true,
			expectDeduperStateContents:    map[string]uint64{"deployment:1": 0},
		},
		"Sensor reconciles: sensor has capability and status is unknown": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{centralsensor.SensorReconciliationOnReconnect},
			givenSensorState:              central.SensorHello_UNKNOWN,
			expectReconciliationMapClosed: true,
			expectDeduperStateSent:        true,
			expectDeduperStateContents:    map[string]uint64{"deployment:1": 0},
		},
		"Sensor reconciles: state is sent even if there is no deduper state": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{centralsensor.SensorReconciliationOnReconnect},
			givenSensorState:              central.SensorHello_RECONNECT,
			expectReconciliationMapClosed: true,
			expectDeduperStateSent:        true,
			expectDeduperStateContents:    nil,
		},
		"Central reconciles: sensor doesn't have capability status is reconnect": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{},
			givenSensorState:              central.SensorHello_RECONNECT,
			expectReconciliationMapClosed: false,
			expectDeduperStateSent:        false,
		},
		"Central reconciles: sensor doesn't have capability status is startup": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{},
			givenSensorState:              central.SensorHello_STARTUP,
			expectReconciliationMapClosed: false,
			expectDeduperStateSent:        false,
		},
		"Central reconciles: sensor doesn't have capability status is unknown": {
			givenSensorCapabilities:       []centralsensor.SensorCapability{},
			givenSensorState:              central.SensorHello_UNKNOWN,
			expectReconciliationMapClosed: false,
			expectDeduperStateSent:        false,
		},
		"Central reconciles: failed to send message": {
			givenSensorCapabilities: []centralsensor.SensorCapability{centralsensor.SensorReconciliationOnReconnect},
			givenSensorState:        central.SensorHello_RECONNECT,
			givenSendError:          errors.New("gRPC error"),
			expectError:             true,
			expectDeduperStateSent:  true,
		},
	}

	for name, tc := range cases {
		s.Run(name, func() {
			ctx := context.Background()

			ctrl := gomock.NewController(s.T())
			mgrMock := clusterMgrMock.NewMockClusterManager(ctrl)
			pipeline := pipelineMock.NewMockClusterPipeline(ctrl)
			deduper := mocks.NewMockDeduper(ctrl)
			stopSig := concurrency.NewErrorSignal()

			hello := &central.SensorHello{
				SensorVersion: "1.0",
				SensorState:   tc.givenSensorState,
			}

			eventHandler := newSensorEventHandler(&storage.Cluster{}, "", pipeline, nil, &stopSig, deduper)

			sensorMockConn := &sensorConnection{
				clusterMgr:         mgrMock,
				sensorEventHandler: eventHandler,
				sensorHello:        hello,
				hashDeduper:        deduper,
			}

			server := &mockServer{
				sentList:          make([]*central.MsgToSensor, 0),
				errOnDeduperState: tc.givenSendError,
			}

			caps := set.NewSet[centralsensor.SensorCapability](tc.givenSensorCapabilities...)

			mgrMock.EXPECT().GetCluster(ctx, gomock.Any()).Return(&storage.Cluster{}, true, nil).AnyTimes()
			if tc.expectDeduperStateSent {
				deduper.EXPECT().GetSuccessfulHashes().Return(tc.expectDeduperStateContents).Times(1)
			} else {
				deduper.EXPECT().GetSuccessfulHashes().Times(0)
			}

			err := sensorMockConn.Run(ctx, server, caps)
			if tc.expectError {
				s.Error(err)
			} else {
				s.NoError(err)
			}

			var deduperState *central.DeduperState
			for _, msg := range server.sentList {
				if m := msg.GetDeduperState(); m != nil {
					deduperState = m
				}
			}

			if tc.expectDeduperStateSent {
				s.NotNil(deduperState)
				s.Equal(tc.expectDeduperStateContents, deduperState.ResourceHashes)
			} else {
				s.Nil(deduperState)
			}

			s.Equal(tc.expectReconciliationMapClosed, eventHandler.reconciliationMap.IsClosed())
		})
	}

}

func (s *testSuite) TestGetPolicySyncMsgFromPoliciesDoesntDowngradeInvalidVersions() {
	sensorMockConn := &sensorConnection{
		sensorHello: &central.SensorHello{
			PolicyVersion: "this ain't a version",
		},
	}

	msg, err := sensorMockConn.getPolicySyncMsgFromPolicies([]*storage.Policy{{PolicyVersion: policyversion.CurrentVersion().String()}})
	s.NoError(err)

	policySync := msg.GetPolicySync()
	s.Require().NotNil(policySync)
	s.NotEmpty(policySync.Policies)
	s.Equal(policyversion.CurrentVersion().String(), policySync.Policies[0].GetPolicyVersion())
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
	caps := set.NewSet(centralsensor.AuditLogEventsCap)

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
	namespace, clusterID, requestID := "namespace", "clusterID", "requestID"
	testCases := map[string]struct {
		requestID  string
		namespace  string
		clusterID  string
		shouldFail bool
	}{
		"no parameter missing": {requestID: requestID, namespace: namespace, clusterID: clusterID, shouldFail: false},
		"requestID missing":    {requestID: "", namespace: namespace, clusterID: clusterID, shouldFail: true},
		"namespace missing":    {requestID: requestID, namespace: "", clusterID: clusterID, shouldFail: true},
		"clusterID missing":    {requestID: requestID, namespace: namespace, clusterID: "", shouldFail: true},
	}
	for tcName, tc := range testCases {
		s.Run(tcName, func() {
			sendC := make(chan *central.MsgToSensor)
			sensorMockConn := &sensorConnection{
				clusterID: tc.clusterID,
				sendC:     sendC,
				stopSig:   concurrency.NewErrorSignal(),
				sensorHello: &central.SensorHello{
					DeploymentIdentification: &storage.SensorDeploymentIdentification{
						AppNamespace: tc.namespace,
					},
				},
			}
			ctx := context.Background()
			ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
			defer cancel()
			request := &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_IssueLocalScannerCertsRequest{
					IssueLocalScannerCertsRequest: &central.IssueLocalScannerCertsRequest{
						RequestId: tc.requestID,
					},
				},
			}

			handleDoneErrSig := concurrency.NewErrorSignal()
			go func() {
				handleDoneErrSig.SignalWithError(sensorMockConn.handleMessage(ctx, request))
			}()

			select {
			case msgToSensor := <-sendC:
				response := msgToSensor.GetIssueLocalScannerCertsResponse()
				s.Equal(tc.requestID, response.GetRequestId())
				if tc.shouldFail {
					s.NotNil(response.GetError())
				} else {
					s.NotNil(response.GetCertificates())
				}
			case <-ctx.Done():
				s.Fail(ctx.Err().Error())
			}

			handleErr, ok := handleDoneErrSig.WaitUntil(ctx)
			s.Require().True(ok)
			s.NoError(handleErr)
		})
	}
}

func (s *testSuite) TestDelegatedRegistryConfigOnRun() {
	ctx := context.Background()
	clusterID := "this-cluster"
	cluster := &storage.Cluster{
		Id: clusterID,
	}

	ctrl := gomock.NewController(s.T())
	mgrMock := clusterMgrMock.NewMockClusterManager(ctrl)
	deleRegMgr := clusterMgrMock.NewMockDelegatedRegistryConfigManager(ctrl)
	iiMgr := clusterMgrMock.NewMockImageIntegrationManager(ctrl)

	sensorMockConn := &sensorConnection{
		clusterID:                  clusterID,
		clusterMgr:                 mgrMock,
		delegatedRegistryConfigMgr: deleRegMgr,
		imageIntegrationMgr:        iiMgr,
	}
	mgrMock.EXPECT().GetCluster(ctx, clusterID).Return(cluster, true, nil).AnyTimes()
	iiMgr.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).AnyTimes()

	s.Run("send", func() {
		caps := set.NewSet(centralsensor.DelegatedRegistryCap)

		config := &storage.DelegatedRegistryConfig{EnabledFor: storage.DelegatedRegistryConfig_ALL}
		deleRegMgr.EXPECT().GetConfig(ctx).Return(config, true, nil)

		server := &mockServer{sentList: make([]*central.MsgToSensor, 0)}
		s.NoError(sensorMockConn.Run(ctx, server, caps))

		for _, msg := range server.sentList {
			if deleConfig := msg.GetDelegatedRegistryConfig(); deleConfig != nil {
				s.Equal(central.DelegatedRegistryConfig_ALL, deleConfig.EnabledFor)
				return
			}
		}

		s.FailNow("Delegated registry config msg was not sent")
	})

	s.Run("no send on no cap", func() {
		caps := set.NewSet[centralsensor.SensorCapability]()

		server := &mockServer{sentList: make([]*central.MsgToSensor, 0)}
		s.NoError(sensorMockConn.Run(ctx, server, caps))

		for _, msg := range server.sentList {
			if deleConfig := msg.GetDelegatedRegistryConfig(); deleConfig != nil {
				s.FailNow("Delegated registry config msg was sent")
				return
			}
		}
	})

	s.Run("no send on nil config", func() {
		caps := set.NewSet(centralsensor.DelegatedRegistryCap)

		deleRegMgr.EXPECT().GetConfig(ctx).Return(nil, false, nil)

		server := &mockServer{sentList: make([]*central.MsgToSensor, 0)}
		s.NoError(sensorMockConn.Run(ctx, server, caps))

		for _, msg := range server.sentList {
			if deleConfig := msg.GetDelegatedRegistryConfig(); deleConfig != nil {
				s.FailNow("Delegated registry config msg was sent")
				return
			}
		}
	})

	s.Run("no send on err", func() {
		caps := set.NewSet(centralsensor.DelegatedRegistryCap)

		deleRegMgr.EXPECT().GetConfig(ctx).Return(nil, false, errors.New("fake error"))

		server := &mockServer{sentList: make([]*central.MsgToSensor, 0)}
		err := sensorMockConn.Run(ctx, server, caps)
		s.ErrorContains(err, "unable to get delegated registry config")
	})
}

func (s *testSuite) TestImageIntegrationsOnRun() {
	ctx := context.Background()
	clusterID := "this-cluster"
	cluster := &storage.Cluster{
		Id: clusterID,
	}

	withCap := set.NewSet(centralsensor.DelegatedRegistryCap)
	withoutCap := set.NewSet[centralsensor.SensorCapability]()
	withRegCategory := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_REGISTRY}
	withoutRegCategory := []storage.ImageIntegrationCategory{storage.ImageIntegrationCategory_SCANNER}

	genServer := func() *mockServer {
		return &mockServer{sentList: make([]*central.MsgToSensor, 0)}
	}

	ctrl := gomock.NewController(s.T())
	mgrMock := clusterMgrMock.NewMockClusterManager(ctrl)
	deleRegMgr := clusterMgrMock.NewMockDelegatedRegistryConfigManager(ctrl)
	iiMgr := clusterMgrMock.NewMockImageIntegrationManager(ctrl)

	sensorMockConn := &sensorConnection{
		clusterID:                  clusterID,
		clusterMgr:                 mgrMock,
		delegatedRegistryConfigMgr: deleRegMgr,
		imageIntegrationMgr:        iiMgr,
	}
	mgrMock.EXPECT().GetCluster(ctx, clusterID).Return(cluster, true, nil).AnyTimes()
	deleRegMgr.EXPECT().GetConfig(ctx).AnyTimes()

	iis := []*storage.ImageIntegration{
		{
			Name: "valid",
			Id:   "id1",
		},
	}

	s.Run("send", func() {
		iis[0].Autogenerated = false
		iis[0].Categories = withRegCategory
		iiMgr.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return(iis, nil)

		server := genServer()
		s.NoError(sensorMockConn.Run(ctx, server, withCap))
		for _, msg := range server.sentList {
			if imgInts := msg.GetImageIntegrations(); imgInts != nil {
				s.Len(imgInts.DeletedIntegrationIds, 0)
				s.Len(imgInts.UpdatedIntegrations, 1)
				s.Equal(imgInts.UpdatedIntegrations[0].Name, "valid")
				s.Equal(imgInts.UpdatedIntegrations[0].Id, "id1")
				return
			}
		}

		s.FailNow("Image integration msg was not sent")
	})

	s.Run("no send on autogenerated", func() {
		iis[0].Autogenerated = true
		iis[0].Categories = withRegCategory
		iiMgr.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return(iis, nil)

		server := genServer()
		s.NoError(sensorMockConn.Run(ctx, server, withCap))
		for _, msg := range server.sentList {
			if imgInts := msg.GetImageIntegrations(); imgInts != nil {
				s.FailNow("Image integrations msg was sent")
				return
			}
		}
	})

	s.Run("no send on no category", func() {
		iis[0].Autogenerated = false
		iis[0].Categories = withoutRegCategory

		iiMgr.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return(iis, nil)

		server := genServer()
		s.NoError(sensorMockConn.Run(ctx, server, withCap))
		for _, msg := range server.sentList {
			if imgInts := msg.GetImageIntegrations(); imgInts != nil {
				s.FailNow("Image integrations msg was sent")
				return
			}
		}
	})

	s.Run("no send on no cap", func() {
		iis[0].Autogenerated = false
		iis[0].Categories = withRegCategory

		server := genServer()
		s.NoError(sensorMockConn.Run(ctx, server, withoutCap))
		for _, msg := range server.sentList {
			if deleConfig := msg.GetDelegatedRegistryConfig(); deleConfig != nil {
				s.FailNow("Image integrations msg was sent")
				return
			}
		}
	})

	s.Run("no send on no integrations", func() {
		iiMgr.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return(nil, nil)

		server := genServer()
		s.NoError(sensorMockConn.Run(ctx, server, withCap))
		for _, msg := range server.sentList {
			if imgInts := msg.GetImageIntegrations(); imgInts != nil {
				s.FailNow("Image integrations msg was sent")
				return
			}
		}
	})

	s.Run("no send on err", func() {
		iiMgr.EXPECT().GetImageIntegrations(gomock.Any(), gomock.Any()).Return(nil, errors.New("broken"))

		server := genServer()
		err := sensorMockConn.Run(ctx, server, withCap)
		s.ErrorContains(err, "unable to get image integrations")
	})
}
