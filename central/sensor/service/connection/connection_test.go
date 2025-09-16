package connection

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"slices"
	"testing"
	"time"

	"github.com/pkg/errors"
	scanConfigMocks "github.com/stackrox/rox/central/complianceoperator/v2/scanconfigurations/datastore/mocks"
	"github.com/stackrox/rox/central/hash/manager/mocks"
	clusterMgrMock "github.com/stackrox/rox/central/sensor/service/common/mocks"
	pipelineMock "github.com/stackrox/rox/central/sensor/service/pipeline/mocks"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/booleanpolicy/policyversion"
	"github.com/stackrox/rox/pkg/centralsensor"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/env"
	"github.com/stackrox/rox/pkg/features"
	testutilsMTLS "github.com/stackrox/rox/pkg/mtls/testutils"
	"github.com/stackrox/rox/pkg/protoassert"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/protoconv/schedule"
	"github.com/stackrox/rox/pkg/sac"
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
	mockCtrl *gomock.Controller

	scanConfigDS *scanConfigMocks.MockDataStore
}

var (
	scanConfigs = []*storage.ComplianceOperatorScanConfigurationV2{
		{
			ScanConfigName: "TestConfigName",
			Profiles: []*storage.ComplianceOperatorScanConfigurationV2_ProfileName{
				{
					ProfileName: "TestProfileName",
				},
			},
			Schedule: &storage.Schedule{
				IntervalType: storage.Schedule_DAILY,
				Hour:         1,
				Minute:       2, Interval: &storage.Schedule_DaysOfWeek_{
					DaysOfWeek: &storage.Schedule_DaysOfWeek{
						Days: []int32{1},
					},
				},
			},
		},
	}
)

func (s *testSuite) SetupTest() {
	err := testutilsMTLS.LoadTestMTLSCerts(s.T())
	s.Require().NoError(err)

	s.mockCtrl = gomock.NewController(s.T())
	s.scanConfigDS = scanConfigMocks.NewMockDataStore(s.mockCtrl)
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

func (s *testSuite) TestSendsScanConfigurationMsgOnRun() {
	// ROX_COMPLIANCE_ENHANCEMENTS is set to 'true' by default but just in case
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	ctx := sac.WithAllAccess(context.Background())

	ctrl := gomock.NewController(s.T())
	mgrMock := clusterMgrMock.NewMockClusterManager(ctrl)
	pipeline := pipelineMock.NewMockClusterPipeline(ctrl)
	deduper := mocks.NewMockDeduper(ctrl)
	stopSig := concurrency.NewErrorSignal()

	hello := &central.SensorHello{
		SensorVersion: "1.0",
	}

	eventHandler := newSensorEventHandler(&storage.Cluster{}, "", pipeline, nil, &stopSig, deduper, nil)

	sensorMockConn := &sensorConnection{
		clusterMgr:         mgrMock,
		sensorEventHandler: eventHandler,
		sensorHello:        hello,
		hashDeduper:        deduper,
		scanSettingDS:      s.scanConfigDS,
		sendC:              make(chan *central.MsgToSensor),
	}

	mgrMock.EXPECT().GetCluster(ctx, gomock.Any()).Return(&storage.Cluster{}, true, nil).Times(2)
	s.scanConfigDS.EXPECT().GetScanConfigurations(ctx, gomock.Any()).Return(scanConfigs, nil).Times(1)

	server := &mockServer{
		sentList: make([]*central.MsgToSensor, 0),
	}

	err := sensorMockConn.Run(ctx, server, set.NewSet[centralsensor.SensorCapability](centralsensor.ComplianceV2ScanConfigSync))
	s.NoError(err)

	var complianceRequests []*central.ComplianceRequest
	for _, msg := range server.sentList {
		if m := msg.GetComplianceRequest(); m != nil {
			complianceRequests = append(complianceRequests, m)
		}
	}

	s.Assert().Equal(1, len(complianceRequests))
	for _, scr := range complianceRequests {
		s.Require().Len(scr.GetSyncScanConfigs().GetScanConfigs(), len(scanConfigs))
		for _, sc := range scr.GetSyncScanConfigs().GetScanConfigs() {
			s.Require().NotNil(sc.GetUpdateScan())
			idx := slices.IndexFunc(scanConfigs, func(slice *storage.ComplianceOperatorScanConfigurationV2) bool {
				return slice.GetScanConfigName() == sc.GetUpdateScan().GetScanSettings().GetScanName()
			})
			s.Require().NotEqual(-1, idx)
			s.Assert().Equal(scanConfigs[idx].GetScanConfigName(), sc.GetUpdateScan().GetScanSettings().GetScanName())
			cron, err := schedule.ConvertToCronTab(scanConfigs[idx].GetSchedule())
			s.Require().NoError(err)
			s.Assert().Equal(cron, sc.GetUpdateScan().GetCron())
		}
	}
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
	s.T().Setenv(features.SensorReconciliationOnReconnect.EnvVar(), "true")
	s.T().Setenv(env.MaxDeduperEntriesPerMessage.EnvVar(), "2")
	if !features.SensorReconciliationOnReconnect.Enabled() {
		s.T().Skip("Test skipped if ROX_SENSOR_RECONCILIATION feature flag isn't set")
	}
	cases := map[string]struct {
		givenSensorCapabilities     []centralsensor.SensorCapability
		givenSensorState            central.SensorHello_SensorState
		givenSendError              error
		expectError                 bool
		expectDeduperStateSent      bool
		expectNumberOfDeduperStates int
		expectDeduperStateContents  map[string]uint64
	}{
		"Sensor reconciles: sensor has capability and status is reconnect": {
			givenSensorCapabilities:     []centralsensor.SensorCapability{centralsensor.SendDeduperStateOnReconnect},
			givenSensorState:            central.SensorHello_RECONNECT,
			expectDeduperStateSent:      true,
			expectNumberOfDeduperStates: 1,
			expectDeduperStateContents:  map[string]uint64{"deployment:1": 0},
		},
		"Sensor reconciles: sensor has capability and status is startup": {
			givenSensorCapabilities:     []centralsensor.SensorCapability{centralsensor.SendDeduperStateOnReconnect},
			givenSensorState:            central.SensorHello_STARTUP,
			expectDeduperStateSent:      true,
			expectNumberOfDeduperStates: 1,
			expectDeduperStateContents:  map[string]uint64{"deployment:1": 0},
		},
		"Sensor reconciles: sensor has capability and status is unknown": {
			givenSensorCapabilities:     []centralsensor.SensorCapability{centralsensor.SendDeduperStateOnReconnect},
			givenSensorState:            central.SensorHello_UNKNOWN,
			expectDeduperStateSent:      true,
			expectNumberOfDeduperStates: 1,
			expectDeduperStateContents:  map[string]uint64{"deployment:1": 0},
		},
		"Sensor reconciles: state is sent even if there is no deduper state": {
			givenSensorCapabilities:     []centralsensor.SensorCapability{centralsensor.SendDeduperStateOnReconnect},
			givenSensorState:            central.SensorHello_RECONNECT,
			expectDeduperStateSent:      true,
			expectNumberOfDeduperStates: 1,
			expectDeduperStateContents:  nil,
		},
		"Central reconciles: sensor doesn't have capability status is reconnect": {
			givenSensorCapabilities: []centralsensor.SensorCapability{},
			givenSensorState:        central.SensorHello_RECONNECT,
			expectDeduperStateSent:  false,
		},
		"Central reconciles: sensor doesn't have capability status is startup": {
			givenSensorCapabilities: []centralsensor.SensorCapability{},
			givenSensorState:        central.SensorHello_STARTUP,
			expectDeduperStateSent:  false,
		},
		"Central reconciles: sensor doesn't have capability status is unknown": {
			givenSensorCapabilities: []centralsensor.SensorCapability{},
			givenSensorState:        central.SensorHello_UNKNOWN,
			expectDeduperStateSent:  false,
		},
		"Central reconciles: failed to send message": {
			givenSensorCapabilities:     []centralsensor.SensorCapability{centralsensor.SendDeduperStateOnReconnect},
			givenSensorState:            central.SensorHello_RECONNECT,
			givenSendError:              errors.New("gRPC error"),
			expectError:                 true,
			expectDeduperStateSent:      true,
			expectNumberOfDeduperStates: 1,
		},
		"Sensor reconciles: multiple chunks are sent": {
			givenSensorCapabilities:     []centralsensor.SensorCapability{centralsensor.SendDeduperStateOnReconnect},
			givenSensorState:            central.SensorHello_RECONNECT,
			expectDeduperStateSent:      true,
			expectNumberOfDeduperStates: 2,
			expectDeduperStateContents:  map[string]uint64{"deployment:1": 0, "deployment:2": 1, "deployment:3": 2},
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

			eventHandler := newSensorEventHandler(&storage.Cluster{}, "", pipeline, nil, &stopSig, deduper, nil)

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

			var deduperStates []*central.DeduperState
			for _, msg := range server.sentList {
				if m := msg.GetDeduperState(); m != nil {
					deduperStates = append(deduperStates, m)
				}
			}

			if tc.expectDeduperStateSent {
				s.NotNil(deduperStates)
				s.Assert().Len(deduperStates, tc.expectNumberOfDeduperStates)
				currentSet := set.NewIntSet()
				for i := 1; i <= tc.expectNumberOfDeduperStates; i++ {
					currentSet.Add(i)
				}
				deduperStateSent := make(map[string]uint64)
				for _, state := range deduperStates {
					for k, v := range state.GetResourceHashes() {
						deduperStateSent[k] = v
					}
					s.Equal(tc.expectNumberOfDeduperStates, int(state.GetTotal()))
					s.True(currentSet.Contains(int(state.GetCurrent())))
					currentSet.Remove(int(state.GetCurrent()))
				}
				if tc.expectDeduperStateContents != nil {
					s.Equal(tc.expectDeduperStateContents, deduperStateSent)
				} else {
					s.Assert().Len(deduperStateSent, 0)
				}
			} else {
				s.Nil(deduperStates)
			}

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
			CollectLogsSince: protocompat.TimestampNow(),
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
			protoassert.MapEqual(s.T(), auditLogState, syncMsg.GetNodeAuditLogFileStates())
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

func (s *testSuite) TestIssueSecuredClusterCerts() {
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
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			request := &central.MsgFromSensor{
				Msg: &central.MsgFromSensor_IssueSecuredClusterCertsRequest{
					IssueSecuredClusterCertsRequest: &central.IssueSecuredClusterCertsRequest{
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
				response := msgToSensor.GetIssueSecuredClusterCertsResponse()
				s.Equal(tc.requestID, response.GetRequestId())
				if tc.shouldFail {
					s.NotNil(response.GetError())
				} else {
					s.NotNil(response.GetCertificates())

					certificates := response.GetCertificates()
					s.NotNil(certificates.GetServiceCerts())

					caPem := certificates.GetCaPem()
					s.NotNil(caPem)
					certBlock, _ := pem.Decode(caPem)
					s.NotNil(certBlock, "Failed to decode CA certificate PEM")
					_, err := x509.ParseCertificate(certBlock.Bytes)
					s.NoError(err, "Invalid CA certificate")

					serviceCertificates := certificates.GetServiceCerts()
					expectedCertificates := 7
					s.Len(serviceCertificates, expectedCertificates, "unexpected number of certificates returned")

					for _, serviceCertificate := range serviceCertificates {
						cert := serviceCertificate.GetCert()

						certPem := cert.GetCertPem()
						keyPem := cert.GetKeyPem()

						s.NotNil(certPem)
						s.NotNil(keyPem)

						certBlock, _ := pem.Decode(certPem)
						s.NotNil(certBlock, "Failed to decode service certificate PEM")
						_, err := x509.ParseCertificate(certBlock.Bytes)
						s.NoError(err, "Invalid service certificate")

						keyBlock, _ := pem.Decode(keyPem)
						s.NotNil(keyBlock, "Failed to decode service key PEM")
						_, err = x509.ParseECPrivateKey(keyBlock.Bytes)
						s.NoError(err, "Invalid service private key PEM format")
					}

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
				s.True(imgInts.Refresh)
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
