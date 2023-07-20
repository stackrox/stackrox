package complianceoperator

import (
	"context"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/k8sutil"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/common/message"
	notifierMocks "github.com/stackrox/rox/sensor/common/mocks"
	"github.com/stackrox/rox/sensor/kubernetes/complianceoperator/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
)

type expectedResponse struct {
	id        string
	errSubstr string
}

func TestManager(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}

type ManagerTestSuite struct {
	suite.Suite

	client         *fake.FakeDynamicClient
	notifier       *notifierMocks.MockNotifier
	requestHandler common.SensorComponent
	statusInfo     *mocks.MockStatusInfo
}

func (s *ManagerTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceEnhancements.EnvVar())
		s.T().SkipNow()
	}
}

func (s *ManagerTestSuite) SetupTest() {
	s.client = fake.NewSimpleDynamicClient(runtime.NewScheme(), &v1alpha1.ScanSettingBinding{TypeMeta: v1.TypeMeta{Kind: "ScanSetting", APIVersion: complianceoperator.GetGroupVersion().String()}})
	s.notifier = notifierMocks.NewMockNotifier(gomock.NewController(s.T()))
	s.statusInfo = mocks.NewMockStatusInfo(gomock.NewController(s.T()))
	s.requestHandler = newTestRequestHandler(s.client, s.statusInfo, s.notifier)
	s.Require().NoError(s.requestHandler.Start())
}

func (s *ManagerTestSuite) TearDownSuite() {
	s.requestHandler.Stop(nil)
}

func (s *ManagerTestSuite) TestProcessApplyOneTimeScanSuccess() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyOneTimeScanInvalid() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "compliance profiles not specified",
	}

	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyOneTimeScanComplianceDisabled() {
	msg := getDisableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}
	s.notifier.EXPECT().Notify(common.SensorComponentEventComplianceDisabled)
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	msg = getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected = expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "Compliance is disabled",
	}
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyOneTimeScanOperatorNSUnknown() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespace not known",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyOneTimeScanAlreadyExists() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	// Retry should fail.
	msg = getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected = expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "\"ad-hoc\" already exists",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyScheduledScanSuccess() {
	msg := getTestScheduledScanRequestMsg("midnight", "* * * * *", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyScheduledScanInvalid() {
	msg := getTestScheduledScanRequestMsg("error", "error")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "compliance profiles not specified, schedule is not valid",
	}

	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyScheduledScanComplianceDisabled() {
	msg := getDisableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}
	s.notifier.EXPECT().Notify(common.SensorComponentEventComplianceDisabled)
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	msg = getTestScheduledScanRequestMsg("midnight", "0 0 * * *", "ocp4-cis")
	expected = expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "Compliance is disabled",
	}
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyScheduledScanOperatorNSUnknown() {
	msg := getTestScheduledScanRequestMsg("midnight", "0 0 * * *", "ocp4-cis")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespace not known",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessApplyScheduledScanAlreadyExists() {
	msg := getTestScheduledScanRequestMsg("midnight", "0 0 * * *", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	// Retry should fail.
	msg = getTestScheduledScanRequestMsg("midnight", "0 0 * * *", "ocp4-cis")
	expected = expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "\"midnight\" already exists",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessDeleteScanConfigSuccess() {
	// create
	msg := getTestScheduledScanRequestMsg("midnight", "0 0 * * *", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	// delete
	msg = getTestDeleteScanConfigMsg("midnight")
	expected = expectedResponse{
		id: msg.GetComplianceRequest().GetDeleteScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessDeleteScanConfigDefaultConfig() {
	msg := getTestDeleteScanConfigMsg(defaultScanSettingName)
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetDeleteScanConfig().GetId(),
		errSubstr: "cannot be deleted",
	}

	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessDeleteScanConfigDisabled() {
	msg := getDisableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}
	s.notifier.EXPECT().Notify(common.SensorComponentEventComplianceDisabled)
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	msg = getTestDeleteScanConfigMsg("fake")
	expected = expectedResponse{
		id:        msg.GetComplianceRequest().GetDeleteScanConfig().GetId(),
		errSubstr: "Compliance is disabled",
	}
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessDeleteScanConfigNotFound() {
	msg := getTestDeleteScanConfigMsg("midnight")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDeleteScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessRerunScanSuccess() {
	// create
	complianceScan := &v1alpha1.ComplianceScan{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSettingGVK.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "midnight",
			Namespace: "ns",
		},
	}
	obj, err := k8sutil.RuntimeObjToUnstructured(complianceScan)
	s.Require().NoError(err)
	_, err = s.client.Resource(complianceoperator.ComplianceScanGVR).Namespace("ns").Create(context.Background(), obj, v1.CreateOptions{})
	s.Require().NoError(err)

	// rerun
	msg := getTestRerunScanMsg("midnight")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestProcessRerunScanNotFound() {
	msg := getTestRerunScanMsg("midnight")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespaces/ns/compliancescans/midnight not found",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) TestEnableDisableCompliance() {
	msg := getTestEnableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetEnableCompliance().GetId(),
	}

	s.notifier.EXPECT().Notify(common.SensorComponentEventComplianceEnabled)
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	msg = getTestDisableComplianceMsg()
	expected = expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}

	s.notifier.EXPECT().Notify(common.SensorComponentEventComplianceDisabled)
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *ManagerTestSuite) sendMessage(times int, msg *central.MsgToSensor) *central.ComplianceResponse {
	timer := time.NewTimer(responseTimeout)
	var ret *central.ComplianceResponse

	for i := 0; i < times; i++ {
		s.NoError(s.requestHandler.ProcessMessage(msg))

		select {
		case response := <-s.requestHandler.ResponsesC():
			ret = response.Msg.(*central.MsgFromSensor_ComplianceResponse).ComplianceResponse
		case <-timer.C:
			s.Fail("Timed out while waiting")
		}
	}
	return ret
}

func (s *ManagerTestSuite) assert(expected expectedResponse, actual *central.ComplianceResponse) {
	var actualID, actualErr string
	switch r := actual.GetResponse().(type) {
	case *central.ComplianceResponse_EnableComplianceResponse_:
		actualID = r.EnableComplianceResponse.GetId()
		actualErr = r.EnableComplianceResponse.GetError()
	case *central.ComplianceResponse_DisableComplianceResponse_:
		actualID = r.DisableComplianceResponse.GetId()
		actualErr = r.DisableComplianceResponse.GetError()
	case *central.ComplianceResponse_ApplyComplianceScanConfigResponse_:
		actualID = r.ApplyComplianceScanConfigResponse.GetId()
		actualErr = r.ApplyComplianceScanConfigResponse.GetError()
	case *central.ComplianceResponse_DeleteComplianceScanConfigResponse_:
		actualID = r.DeleteComplianceScanConfigResponse.GetId()
		actualErr = r.DeleteComplianceScanConfigResponse.GetError()
	}

	s.Equal(expected.id, actualID)
	if expected.errSubstr == "" {
		s.Empty(actualErr)
	} else {
		s.Contains(actualErr, expected.errSubstr)
	}
}

func getTestOneTimeScanRequestMsg(name string, profiles ...string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_ApplyScanConfig{
					ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
						Id: uuid.NewV4().String(),
						ScanRequest: &central.ApplyComplianceScanConfigRequest_OneTimeScan_{
							OneTimeScan: &central.ApplyComplianceScanConfigRequest_OneTimeScan{
								ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
									ScanName:       name,
									StrictNodeScan: true,
									Profiles:       profiles,
								},
							},
						},
					},
				},
			},
		},
	}
}

func getTestScheduledScanRequestMsg(name, cron string, profiles ...string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_ApplyScanConfig{
					ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
						Id: uuid.NewV4().String(),
						ScanRequest: &central.ApplyComplianceScanConfigRequest_ScheduledScan_{
							ScheduledScan: &central.ApplyComplianceScanConfigRequest_ScheduledScan{
								ScanSettings: &central.ApplyComplianceScanConfigRequest_BaseScanSettings{
									ScanName:       name,
									StrictNodeScan: true,
									Profiles:       profiles,
								},
								Cron: cron,
							},
						},
					},
				},
			},
		},
	}
}

func getDisableComplianceMsg() *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_DisableCompliance{
					DisableCompliance: &central.DisableComplianceRequest{
						Id: uuid.NewV4().String(),
					},
				},
			},
		},
	}
}

func getTestDeleteScanConfigMsg(name string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_DeleteScanConfig{
					DeleteScanConfig: &central.DeleteComplianceScanConfigRequest{
						Id:   uuid.NewV4().String(),
						Name: name,
					},
				},
			},
		},
	}
}

func getTestRerunScanMsg(name string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_ApplyScanConfig{
					ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
						Id: uuid.NewV4().String(),
						ScanRequest: &central.ApplyComplianceScanConfigRequest_RerunScan{
							RerunScan: &central.ApplyComplianceScanConfigRequest_RerunScheduledScan{
								ScanName: name,
							},
						},
					},
				},
			},
		},
	}
}

func getTestEnableComplianceMsg() *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_EnableCompliance{
					EnableCompliance: &central.EnableComplianceRequest{
						Id: uuid.NewV4().String(),
					},
				},
			},
		},
	}
}

func getTestDisableComplianceMsg() *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_DisableCompliance{
					DisableCompliance: &central.DisableComplianceRequest{
						Id: uuid.NewV4().String(),
					},
				},
			},
		},
	}
}

// Test Handler has compliance enabled by default to make testing easier.
func newTestRequestHandler(client dynamic.Interface, complianceOperatorInfo StatusInfo, notifiers ...common.Notifier) common.SensorComponent {
	h := &handlerImpl{
		client:                 client,
		complianceOperatorInfo: complianceOperatorInfo,
		notifiers:              notifiers,
		request:                make(chan *central.ComplianceRequest),
		response:               make(chan *message.ExpiringMessage),

		disabled:   concurrency.NewSignal(),
		stopSignal: concurrency.NewSignal(),
	}

	return h
}
