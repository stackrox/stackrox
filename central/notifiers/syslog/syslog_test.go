package syslog

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stackrox/rox/central/notifiers/syslog/mocks"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/fixtures"
	metadataGetterMocks "github.com/stackrox/rox/pkg/notifiers/mocks"
	"github.com/stackrox/rox/pkg/protocompat"
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
		maxMessageSize: 32768,
	}
}

func makeNotifier() *storage.Notifier {
	return storage.Notifier_builder{
		Id:   "testID",
		Name: "testName",
		Type: "syslog",
		Syslog: storage.Syslog_builder{
			TcpConfig: storage.Syslog_TCPConfig_builder{
				Hostname: "hostname",
			}.Build(),
			MessageFormat: storage.Syslog_CEF,
		}.Build(),
	}.Build()
}

func makeNotifierExtrafields(keyVals []*storage.KeyValuePair) *storage.Notifier {
	return storage.Notifier_builder{
		Id:   "testID",
		Name: "testName",
		Type: "syslog",
		Syslog: storage.Syslog_builder{
			TcpConfig: storage.Syslog_TCPConfig_builder{
				Hostname: "hostname",
			}.Build(),
			MessageFormat: storage.Syslog_CEF,
			ExtraFields:   keyVals,
		}.Build(),
	}.Build()
}

func (s *SyslogNotifierTestSuite) setupMockMetadataGetterForAlert(alert *storage.Alert) {
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
	value := time.Now()

	msTs := int64(value.Unix())*1000 + int64(value.Nanosecond())/1000000
	expectedValue := []string{fmt.Sprintf("%s=%s", key, strconv.Itoa(int(msTs)))}

	extensionPair := makeTimestampExtensionPair(key, &value)
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
	kvp := &storage.KeyValuePair{}
	kvp.SetKey("")
	kvp.SetValue("")
	keyVals := []*storage.KeyValuePair{kvp}

	notifier := makeNotifierExtrafields(keyVals)
	sys := notifier.GetSyslog()
	e := validateSyslog(sys)
	s.ErrorContains(e, "all extra fields must have both a key and a value")
}

func (s *SyslogNotifierTestSuite) TestValidateSyslogExtraFieldsEmptyList() {
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
	kvp := &storage.KeyValuePair{}
	kvp.SetKey("foo")
	kvp.SetValue("bar")
	keyVals := []*storage.KeyValuePair{kvp}
	notifier := makeNotifierExtrafields(keyVals)
	sys := notifier.GetSyslog()
	e := validateSyslog(sys)
	s.NoError(e)
}

func (s *SyslogNotifierTestSuite) TestValidateAlertToCEFWithExtraFields() {
	kvp := &storage.KeyValuePair{}
	kvp.SetKey("foo")
	kvp.SetValue("bar")
	keyVals := []*storage.KeyValuePair{kvp}
	notifier := makeNotifierExtrafields(keyVals)
	testAlert := fixtures.GetAlert()
	s.setupMockMetadataGetterForAlert(testAlert)
	a := s.makeSyslog(notifier).alertToCEF(context.Background(), testAlert)
	s.Contains(a, "foo=bar")
}

func (s *SyslogNotifierTestSuite) TestValidateAlertToCEFWithNamespaceLabels() {
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
	kvp := &storage.KeyValuePair{}
	kvp.SetKey("foo")
	kvp.SetValue("bar")
	keyVals := []*storage.KeyValuePair{kvp}
	notifier := makeNotifierExtrafields(keyVals)
	userInfo := &storage.UserInfo{}
	userInfo.SetUsername("Joseph")
	userInfo.SetFriendlyName("Rules")
	userInfo.ClearPermissions()
	userInfo.SetRoles(nil)
	amr := &v1.Audit_Message_Request{}
	amr.SetEndpoint("asg")
	amr.SetMethod("jtyr")
	amr.ClearPayload()
	testAuditMessage := &v1.Audit_Message{}
	testAuditMessage.SetTime(protocompat.TimestampNow())
	testAuditMessage.SetUser(userInfo)
	testAuditMessage.SetRequest(amr)
	syslog := s.makeSyslog(notifier)

	m := syslog.auditLogToCEF(testAuditMessage, notifier)
	s.Contains(m, "foo=bar")
}

func (s *SyslogNotifierTestSuite) TestSendAuditLog() {
	notifier := makeNotifier()
	syslog := s.makeSyslog(notifier)
	userInfo := &storage.UserInfo{}
	userInfo.SetUsername("Joseph")
	userInfo.SetFriendlyName("Rules")
	userInfo.ClearPermissions()
	userInfo.SetRoles(nil)
	amr := &v1.Audit_Message_Request{}
	amr.SetEndpoint("asg")
	amr.SetMethod("jtyr")
	amr.ClearPayload()
	testAuditMessage := &v1.Audit_Message{}
	testAuditMessage.SetTime(protocompat.TimestampNow())
	testAuditMessage.SetUser(userInfo)
	testAuditMessage.SetRequest(amr)

	s.mockSender.EXPECT().SendSyslog(gomock.Any()).Return(nil)
	err := syslog.SendAuditMessage(context.Background(), testAuditMessage)
	s.Require().NoError(err)
}

func (s *SyslogNotifierTestSuite) TestAlerts() {
	syslog := s.makeSyslog(makeNotifier())
	testAlert := fixtures.GetAlert()
	s.mockSender.EXPECT().SendSyslog(gomock.Any()).Return(nil).AnyTimes()
	s.setupMockMetadataGetterForAlert(testAlert)
	s.Require().NoError(syslog.AlertNotify(context.Background(), testAlert))

	// Ensure it doesn't panic with nil timestamps
	testAlert.ClearFirstOccurred()
	s.setupMockMetadataGetterForAlert(testAlert)
	s.Require().NoError(syslog.AlertNotify(context.Background(), testAlert))
}

func (s *SyslogNotifierTestSuite) TestValidateRemoteConfig() {
	tcpConfig := &storage.Syslog_TCPConfig{}
	tcpConfig.SetHostname("google.com")
	tcpConfig.SetPort(66666666)
	tcpConfig.SetSkipTlsVerify(true)
	tcpConfig.SetUseTls(true)
	_, errPort := validateRemoteConfig(tcpConfig)
	s.Error(errPort)

	tcpConfigExtraSpace := &storage.Syslog_TCPConfig{}
	tcpConfigExtraSpace.SetHostname("10.46.152.34 ")
	tcpConfigExtraSpace.SetPort(514)
	tcpConfigExtraSpace.SetSkipTlsVerify(true)
	tcpConfigExtraSpace.SetUseTls(true)
	_, errURL := validateRemoteConfig(tcpConfigExtraSpace)
	s.Error(errURL)

	tcpConfigValidIP := &storage.Syslog_TCPConfig{}
	tcpConfigValidIP.SetHostname("10.46.152.34")
	tcpConfigValidIP.SetPort(514)
	tcpConfigValidIP.SetSkipTlsVerify(true)
	tcpConfigValidIP.SetUseTls(true)
	_, errURLValidIP := validateRemoteConfig(tcpConfigValidIP)
	s.NoError(errURLValidIP)
}

func (s *SyslogNotifierTestSuite) TestHeaderFormat() {
	notifier := makeNotifier()
	syslog := s.makeSyslog(notifier)
	header := syslog.getCEFHeaderWithExtension("deviceEventClassID", "alertnameunique", 999, "extension")
	s.Regexp(`^CEF:0\|StackRox\|Kubernetes Security Platform\|.*\|deviceEventClassID\|alertnameunique\|999\|extension$`, header)

	// Legacy format
	syslog.Notifier.GetSyslog().SetMessageFormat(storage.Syslog_LEGACY)
	header = syslog.getCEFHeaderWithExtension("deviceEventClassID", "alertnameunique", 999, "extension")
	s.Regexp(`^CEF:0\|StackRox\|Kubernetes Security Platform\|.*\|deviceEventClassID\|999\|alertnameunique\|extension$`, header)
}
