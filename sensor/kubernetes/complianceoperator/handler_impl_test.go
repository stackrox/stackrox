package complianceoperator

import (
	"context"
	"testing"
	"time"

	"github.com/ComplianceAsCode/compliance-operator/pkg/apis/compliance/v1alpha1"
	"github.com/stackrox/rox/generated/internalapi/central"
	"github.com/stackrox/rox/pkg/complianceoperator"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stackrox/rox/sensor/common"
	"github.com/stackrox/rox/sensor/kubernetes/complianceoperator/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
)

type expectedResponse struct {
	id        string
	errSubstr string
}

func TestHandler(t *testing.T) {
	suite.Run(t, new(HandlerTestSuite))
}

type HandlerTestSuite struct {
	suite.Suite

	client         *fake.FakeDynamicClient
	requestHandler common.SensorComponent
	statusInfo     *mocks.MockStatusInfo
}

func (s *HandlerTestSuite) SetupSuite() {
	s.T().Setenv(features.ComplianceEnhancements.EnvVar(), "true")

	if !features.ComplianceEnhancements.Enabled() {
		s.T().Skipf("Skipping because %s=false", features.ComplianceEnhancements.EnvVar())
		s.T().SkipNow()
	}
}

func (s *HandlerTestSuite) SetupTest() {
	s.client = fake.NewSimpleDynamicClient(runtime.NewScheme(), &v1alpha1.ScanSettingBinding{TypeMeta: v1.TypeMeta{Kind: "ScanSetting", APIVersion: complianceoperator.GetGroupVersion().String()}})
	s.statusInfo = mocks.NewMockStatusInfo(gomock.NewController(s.T()))
	s.requestHandler = NewRequestHandler(s.client, s.statusInfo)
	s.Require().NoError(s.requestHandler.Start())
}

func (s *HandlerTestSuite) TearDownSuite() {
	s.requestHandler.Stop(nil)
}

func (s *HandlerTestSuite) TestProcessApplyOneTimeScanSuccess() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessApplyOneTimeScanInvalid() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "compliance profiles not specified",
	}

	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessApplyOneTimeScanComplianceDisabled() {
	msg := getDisableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}
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

func (s *HandlerTestSuite) TestProcessApplyOneTimeScanOperatorNSUnknown() {
	msg := getTestOneTimeScanRequestMsg("ad-hoc", "ocp4-cis")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespace not known",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessApplyOneTimeScanAlreadyExists() {
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

func (s *HandlerTestSuite) TestProcessApplyScheduledScanSuccess() {
	msg := getTestScheduledScanRequestMsg("midnight", "* * * * *", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessApplyScheduledScanInvalid() {
	msg := getTestScheduledScanRequestMsg("error", "error")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "compliance profiles not specified, schedule is not valid",
	}

	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessApplyScheduledScanComplianceDisabled() {
	msg := getDisableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}
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

func (s *HandlerTestSuite) TestProcessApplyScheduledScanOperatorNSUnknown() {
	msg := getTestScheduledScanRequestMsg("midnight", "0 0 * * *", "ocp4-cis")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespace not known",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessApplyScheduledScanAlreadyExists() {
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

func (s *HandlerTestSuite) TestProcessDeleteScanConfigSuccess() {
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

func (s *HandlerTestSuite) TestProcessDeleteScanConfigDefaultConfig() {
	msg := getTestDeleteScanConfigMsg(defaultScanSettingName)
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetDeleteScanConfig().GetId(),
		errSubstr: "cannot be deleted",
	}

	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessDeleteScanConfigDisabled() {
	msg := getDisableComplianceMsg()
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDisableCompliance().GetId(),
	}
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

func (s *HandlerTestSuite) TestProcessDeleteScanConfigNotFound() {
	msg := getTestDeleteScanConfigMsg("midnight")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetDeleteScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessRerunScanSuccess() {
	// create
	complianceScan := &v1alpha1.ComplianceScan{
		TypeMeta: v1.TypeMeta{
			Kind:       complianceoperator.ScanSetting.Kind,
			APIVersion: complianceoperator.GetGroupVersion().String(),
		},
		ObjectMeta: v1.ObjectMeta{
			Name:      "midnight",
			Namespace: "ns",
		},
	}
	obj, err := runtimeObjToUnstructured(complianceScan)
	s.Require().NoError(err)
	_, err = s.client.Resource(complianceoperator.ComplianceScan.GroupVersionResource()).
		Namespace("ns").Create(context.Background(), obj, v1.CreateOptions{})
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

func (s *HandlerTestSuite) TestProcessRerunScanNotFound() {
	msg := getTestRerunScanMsg("midnight")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespaces/ns/compliancescans/midnight not found",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessSuspendingScanNotFound() {
	msg := getTestSuspendScanMsg("midnight")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespaces/ns/scansettings/midnight not found",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessSuspendResumingScanSuccess() {
	// create a scheduled scan
	msg := getTestScheduledScanRequestMsg("midnight", "* * * * *", "ocp4-cis")
	expected := expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)

	// suspend
	msg = getTestSuspendScanMsg("midnight")
	expected = expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)

	// resume
	msg = getTestResumeScanMsg("midnight")
	expected = expectedResponse{
		id: msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual = s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) TestProcessResumingScanNotFound() {
	msg := getTestResumeScanMsg("midnight")
	expected := expectedResponse{
		id:        msg.GetComplianceRequest().GetApplyScanConfig().GetId(),
		errSubstr: "namespaces/ns/scansettings/midnight not found",
	}

	s.statusInfo.EXPECT().GetNamespace().Return("ns")
	actual := s.sendMessage(1, msg)
	s.assert(expected, actual)
}

func (s *HandlerTestSuite) sendMessage(times int, msg *central.MsgToSensor) *central.ComplianceResponse {
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

func (s *HandlerTestSuite) assert(expected expectedResponse, actual *central.ComplianceResponse) {
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

func getTestSuspendScanMsg(name string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_ApplyScanConfig{
					ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
						Id: uuid.NewV4().String(),
						ScanRequest: &central.ApplyComplianceScanConfigRequest_SuspendScan{
							SuspendScan: &central.ApplyComplianceScanConfigRequest_SuspendScheduledScan{
								ScanName: name,
							},
						},
					},
				},
			},
		},
	}
}

func getTestResumeScanMsg(name string) *central.MsgToSensor {
	return &central.MsgToSensor{
		Msg: &central.MsgToSensor_ComplianceRequest{
			ComplianceRequest: &central.ComplianceRequest{
				Request: &central.ComplianceRequest_ApplyScanConfig{
					ApplyScanConfig: &central.ApplyComplianceScanConfigRequest{
						Id: uuid.NewV4().String(),
						ScanRequest: &central.ApplyComplianceScanConfigRequest_ResumeScan{
							ResumeScan: &central.ApplyComplianceScanConfigRequest_ResumeScheduledScan{
								ScanName: name,
							},
						},
					},
				},
			},
		},
	}
}
