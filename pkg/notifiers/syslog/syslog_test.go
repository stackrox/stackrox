package syslog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/golang/mock/gomock"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	"github.com/stackrox/rox/pkg/notifiers/syslog/mocks"
	"github.com/stretchr/testify/suite"
)

func TestSyslogNotifier(t *testing.T) {
	suite.Run(t, new(SyslogNotifierTestSuite))
}

type SyslogNotifierTestSuite struct {
	suite.Suite

	mockCtrl   *gomock.Controller
	mockSender *mocks.MocksyslogSender
}

func (s *SyslogNotifierTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.mockSender = mocks.NewMocksyslogSender(s.mockCtrl)
	s.T().Setenv(features.RoxSyslogExtraFields.EnvVar(), "true")
}

func (s *SyslogNotifierTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SyslogNotifierTestSuite) makeSyslog(notifier *storage.Notifier) *syslog {
	return &syslog{
		Notifier: notifier,
		sender:   s.mockSender,
		pid:      1,
		facility: (int(notifier.GetSyslog().GetLocalFacility()) + 16) * 8,
	}
}

func makeNotifier() *storage.Notifier {
	return &storage.Notifier{
		Id:   "testID",
		Name: "testName",
		Type: "syslog",
		Config: &storage.Notifier_Syslog{
			Syslog: &storage.Syslog{
				Endpoint: &storage.Syslog_TcpConfig{
					TcpConfig: &storage.Syslog_TCPConfig{
						Hostname: "hostname",
					},
				},
			},
		},
	}
}

func makeNotifierExtrafields(keyVals []*storage.KeyValuePair) *storage.Notifier {
	return &storage.Notifier{
		Id:   "testID",
		Name: "testName",
		Type: "syslog",
		Config: &storage.Notifier_Syslog{
			Syslog: &storage.Syslog{
				Endpoint: &storage.Syslog_TcpConfig{
					TcpConfig: &storage.Syslog_TCPConfig{
						Hostname: "hostname",
					},
				},
				ExtraFields: keyVals,
			},
		},
	}
}

func (s *SyslogNotifierTestSuite) TestCEFMakeExtensionPair() {
	key := "key1"
	value := "value1"

	extensionPair := makeExtensionPair(key, value)
	s.Equal(fmt.Sprintf("%s=%s", key, value), extensionPair)
}

func (s *SyslogNotifierTestSuite) TestCEFMakeJSONExtensionPair() {
	key := "key"
	value := makeNotifier()

	jsonBytes, err := json.Marshal(value)
	s.Require().NoError(err)
	jsonValue := jsonBytes
	expectedExtensionPair := fmt.Sprintf("%s=%s", key, jsonValue)

	extensionPair := makeJSONExtensionPair(key, value)
	s.Equal(expectedExtensionPair, extensionPair)
}

func (s *SyslogNotifierTestSuite) TestCEFMakeTimestampExtensionPair() {
	key := "key"
	value := types.TimestampNow()

	msTs := int64(value.GetSeconds())*1000 + int64(value.GetNanos())/1000000
	expectedValue := []string{fmt.Sprintf("%s=%s", key, strconv.Itoa(int(msTs)))}

	extensionPair := makeTimestampExtensionPair(key, value)
	s.Equal(expectedValue, extensionPair)
}

func (s *SyslogNotifierTestSuite) TestCEFExtensionFromPairs() {
	extensionPair1 := makeExtensionPair("key1", "value1")
	extensionPair2 := makeExtensionPair("key2", "value2")

	extensionPairs := []string{extensionPair1, extensionPair2}
	extension := makeExtensionFromPairs(extensionPairs)
	s.Equal(fmt.Sprintf("%s %s", extensionPair1, extensionPair2), extension)
}

func (s *SyslogNotifierTestSuite) TestValidateSyslogEmptyExtrafields() {
	if !features.RoxSyslogExtraFields.Enabled() {
		s.T().Skip("Skip syslog extra fields tests")
		s.T().SkipNow()
	}

	keyVals := []*storage.KeyValuePair{{Key: "", Value: ""}}

	notifier := makeNotifierExtrafields(keyVals)
	sys := notifier.GetSyslog()
	e := validateSyslog(sys)
	s.ErrorContains(e, "all extra fields must have both a key and a value")
}

func (s *SyslogNotifierTestSuite) TestValidateSyslogExtraFieldsEmptyList() {
	if !features.RoxSyslogExtraFields.Enabled() {
		s.T().Skip("Skip syslog extra fields tests")
		s.T().SkipNow()
	}

	keyVals := []*storage.KeyValuePair{}
	notifier := makeNotifierExtrafields(keyVals)

	sys := notifier.GetSyslog()
	e := validateSyslog(sys)
	s.NoError(e)

	testAlert := fixtures.GetAlert()
	a := alertToCEF(testAlert, notifier)
	s.NotEmpty(a)
}

func (s *SyslogNotifierTestSuite) TestValidateSyslogExtraFields() {
	if !features.RoxSyslogExtraFields.Enabled() {
		s.T().Skip("Skip syslog extra fields tests")
		s.T().SkipNow()
	}

	keyVals := []*storage.KeyValuePair{{Key: "foo", Value: "bar"}}
	notifier := makeNotifierExtrafields(keyVals)
	sys := notifier.GetSyslog()
	e := validateSyslog(sys)
	s.NoError(e)
}

func (s *SyslogNotifierTestSuite) TestValidateAlerttoCef() {
	if !features.RoxSyslogExtraFields.Enabled() {
		s.T().Skip("Skip syslog extra fields tests")
		s.T().SkipNow()
	}

	keyVals := []*storage.KeyValuePair{{Key: "foo", Value: "bar"}}
	notifier := makeNotifierExtrafields(keyVals)
	testAlert := fixtures.GetAlert()
	a := alertToCEF(testAlert, notifier)
	s.Contains(a, "foo=bar")
}

func (s *SyslogNotifierTestSuite) TestValidateExtraFieldsAuditLog() {
	if !features.RoxSyslogExtraFields.Enabled() {
		s.T().Skip("Skip syslog extra fields tests")
		s.T().SkipNow()
	}

	keyVals := []*storage.KeyValuePair{{Key: "foo", Value: "bar"}}
	notifier := makeNotifierExtrafields(keyVals)
	testAuditMessage := &v1.Audit_Message{
		Time: types.TimestampNow(),
		User: &storage.UserInfo{
			Username:     "Joseph",
			FriendlyName: "Rules",
			Permissions:  nil,
			Roles:        nil,
		},
		Request: &v1.Audit_Message_Request{
			Endpoint: "asg",
			Method:   "jtyr",
			Payload:  nil,
		},
	}

	m := auditLogToCEF(testAuditMessage, notifier)
	s.Contains(m, "foo=bar")
}

func (s *SyslogNotifierTestSuite) TestSendAuditLog() {
	notifier := makeNotifier()
	syslog := s.makeSyslog(notifier)
	testAuditMessage := &v1.Audit_Message{
		Time: types.TimestampNow(),
		User: &storage.UserInfo{
			Username:     "Joseph",
			FriendlyName: "Rules",
			Permissions:  nil,
			Roles:        nil,
		},
		Request: &v1.Audit_Message_Request{
			Endpoint: "asg",
			Method:   "jtyr",
			Payload:  nil,
		},
	}

	s.mockSender.EXPECT().SendSyslog(gomock.Any()).Return(nil)
	err := syslog.SendAuditMessage(context.Background(), testAuditMessage)
	s.Require().NoError(err)
}

func (s *SyslogNotifierTestSuite) TestAlerts() {
	syslog := s.makeSyslog(makeNotifier())
	testAlert := fixtures.GetAlert()
	s.mockSender.EXPECT().SendSyslog(gomock.Any()).Return(nil)
	s.Require().NoError(syslog.AlertNotify(context.Background(), testAlert))

	// Ensure it doesn't panic with nil timestamps
	testAlert.FirstOccurred = nil
	s.mockSender.EXPECT().SendSyslog(gomock.Any()).Return(nil)
	s.Require().NoError(syslog.AlertNotify(context.Background(), testAlert))
}

func (s *SyslogNotifierTestSuite) TestValidateRemoteConfig() {
	tcpConfig := &storage.Syslog_TCPConfig{
		Hostname:      "google.com",
		Port:          66666666,
		SkipTlsVerify: true,
		UseTls:        true,
	}
	_, errPort := validateRemoteConfig(tcpConfig)
	s.Error(errPort)

	tcpConfigExtraSpace := &storage.Syslog_TCPConfig{
		Hostname:      "10.46.152.34 ",
		Port:          514,
		SkipTlsVerify: true,
		UseTls:        true,
	}
	_, errURL := validateRemoteConfig(tcpConfigExtraSpace)
	s.Error(errURL)

	tcpConfigValidIP := &storage.Syslog_TCPConfig{
		Hostname:      "10.46.152.34",
		Port:          514,
		SkipTlsVerify: true,
		UseTls:        true,
	}
	_, errURLValidIP := validateRemoteConfig(tcpConfigValidIP)
	s.NoError(errURLValidIP)
}
