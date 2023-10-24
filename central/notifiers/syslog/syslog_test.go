package syslog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/central/notifiers/syslog/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/features"
	"github.com/stackrox/rox/pkg/fixtures"
	metadataGetterMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestSyslogNotifier(t *testing.T) {
	suite.Run(t, new(SyslogNotifierTestSuite))
}

type SyslogNotifierTestSuite struct {
	suite.Suite

	mockCtrl           *gomock.Controller
	mockSender         *mocks.MocksyslogSender
	mockMetadataGetter *metadataGetterMocks.MockMetadataGetter
}

func (s *SyslogNotifierTestSuite) SetupTest() {
	s.mockCtrl = gomock.NewController(s.T())

	s.mockSender = mocks.NewMocksyslogSender(s.mockCtrl)
	s.mockMetadataGetter = metadataGetterMocks.NewMockMetadataGetter(s.mockCtrl)
	s.T().Setenv(features.RoxSyslogExtraFields.EnvVar(), "true")
	s.T().Setenv(features.SyslogNamespaceLabels.EnvVar(), "true")
}

func (s *SyslogNotifierTestSuite) TearDownTest() {
	s.mockCtrl.Finish()
}

func (s *SyslogNotifierTestSuite) makeSyslog(notifier *storage.Notifier) *syslog {
	return &syslog{
		Notifier:       notifier,
		metadataGetter: s.mockMetadataGetter,
		sender:         s.mockSender,
		pid:            1,
		facility:       (int(notifier.GetSyslog().GetLocalFacility()) + 16) * 8,
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
				MessageFormat: storage.Syslog_CEF,
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
				MessageFormat: storage.Syslog_CEF,
				ExtraFields:   keyVals,
			},
		},
	}
}

func (s *SyslogNotifierTestSuite) setupMockMetadataGetterForAlert(alert *storage.Alert) {
	if !features.SyslogNamespaceLabels.Enabled() {
		// No calls to GetNamespaceLabels expected if ROX_SEND_NAMESPACE_LABELS_IN_SYSLOG is disabled
		return
	}

	s.mockMetadataGetter.EXPECT().GetNamespaceLabels(gomock.Any(), alert).Return(map[string]string{
		"x":                           "y",
		"abc":                         "xyz",
		"kubernetes.io/metadata.name": "stackrox",
	})
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
	s.setupMockMetadataGetterForAlert(testAlert)
	a := s.makeSyslog(notifier).alertToCEF(context.Background(), testAlert)
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

func (s *SyslogNotifierTestSuite) TestValidateAlertToCEFWithExtraFields() {
	if !features.RoxSyslogExtraFields.Enabled() {
		s.T().Skip("Skip syslog extra fields tests")
		s.T().SkipNow()
	}

	keyVals := []*storage.KeyValuePair{{Key: "foo", Value: "bar"}}
	notifier := makeNotifierExtrafields(keyVals)
	testAlert := fixtures.GetAlert()
	s.setupMockMetadataGetterForAlert(testAlert)
	a := s.makeSyslog(notifier).alertToCEF(context.Background(), testAlert)
	s.Contains(a, "foo=bar")
}

func (s *SyslogNotifierTestSuite) TestValidateAlertToCEFWithNamespaceLabels() {
	if !features.SyslogNamespaceLabels.Enabled() {
		s.T().Skip("Skipping since ROX_SEND_NAMESPACE_LABELS_IN_SYSLOG is not enabled")
		s.T().SkipNow()
	}

	cases := []struct {
		title                  string
		alert                  *storage.Alert
		namespaceInNotifcation bool
		expectedNamespaceProp  string
		expectedLabelsProp     string
	}{
		{
			title:                  "Namespace and labels should be in included for deployment alert",
			alert:                  fixtures.GetScopedDeploymentAlert("xyz", "cluster-id", "deployment-namespace"),
			namespaceInNotifcation: true,
			expectedNamespaceProp:  "ns=deployment-namespace",
			expectedLabelsProp:     "nslabels={\"abc\":\"xyz\",\"kubernetes.io/metadata.name\":\"stackrox\",\"x\":\"y\"}",
		},
		{
			title:                  "Namespace and labels should be in included for resource alert",
			alert:                  fixtures.GetScopedResourceAlert("abcd", "cluser-id", "my-namespace"),
			namespaceInNotifcation: true,
			expectedNamespaceProp:  "ns=my-namespace",
			expectedLabelsProp:     "nslabels={\"abc\":\"xyz\",\"kubernetes.io/metadata.name\":\"stackrox\",\"x\":\"y\"}",
		},
		{
			title: "Namespace and labels should not be in included for image alert",
			alert: fixtures.GetImageAlert(),
		},
		{
			title: "Namespace and labels should not be in included for alert that's missing a namespace",
			alert: fixtures.GetScopedDeploymentAlert("xyz", "cluster-id", ""),
		},
	}

	notifier := makeNotifier()

	for _, c := range cases {
		s.T().Run(c.title, func(t *testing.T) {
			if c.namespaceInNotifcation {
				s.setupMockMetadataGetterForAlert(c.alert)
			}
			cef := s.makeSyslog(notifier).alertToCEF(context.Background(), c.alert)
			if c.namespaceInNotifcation {
				s.Contains(cef, c.expectedNamespaceProp)
				s.Contains(cef, c.expectedLabelsProp)
			}
		})
	}
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
	syslog := s.makeSyslog(notifier)

	m := syslog.auditLogToCEF(testAuditMessage, notifier)
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
	s.setupMockMetadataGetterForAlert(testAlert)
	s.Require().NoError(syslog.AlertNotify(context.Background(), testAlert))

	// Ensure it doesn't panic with nil timestamps
	testAlert.FirstOccurred = nil
	s.mockSender.EXPECT().SendSyslog(gomock.Any()).Return(nil)
	s.setupMockMetadataGetterForAlert(testAlert)
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

func (s *SyslogNotifierTestSuite) TestHeaderFormat() {
	notifier := makeNotifier()
	syslog := s.makeSyslog(notifier)
	header := syslog.getCEFHeaderWithExtension("deviceEventClassID", "alertnameunique", 999, "extension")
	s.Regexp(`^CEF:0\|StackRox\|Kubernetes Security Platform\|.*\|deviceEventClassID\|alertnameunique\|999\|extension$`, header)

	// Legacy format
	syslog.Notifier.GetSyslog().MessageFormat = storage.Syslog_LEGACY
	header = syslog.getCEFHeaderWithExtension("deviceEventClassID", "alertnameunique", 999, "extension")
	s.Regexp(`^CEF:0\|StackRox\|Kubernetes Security Platform\|.*\|deviceEventClassID\|999\|alertnameunique\|extension$`, header)
}
