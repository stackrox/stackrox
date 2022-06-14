package syslog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/central/notifiers"
	v1 "github.com/stackrox/rox/generated/api/v1"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/logging"
	"github.com/stackrox/rox/pkg/version"
)

const (
	syslogVersion = 1                                    // Currently seems to be the only version in use
	application   = "stackRoxKubernetesSecurityPlatform" // Application name for the syslog header

	deviceReceiptTime                          = "rt"
	sourceUserPrivileges                       = "spriv"
	sourceUserName                             = "suser"
	requestURL                                 = "request"
	requestMethod                              = "requestMethod"
	eventOutcome                               = "outcome"
	requestClientApplication                   = "requestClientApplication"
	stackroxKubernetesSecurityPlatformAuditLog = "stackroxKubernetesSecurityPlatformAuditLog"

	devicePayloadID                         = "devicePayloadId"
	startTime                               = "start"
	endTime                                 = "end"
	stackroxKubernetesSecurityPlatformAlert = "stackroxKubernetesSecurityPlatformAlert"

	// This happens to be the same string as the stackroxKubernetesSecurityPlatformAlert const but that's an accident.
	// stackroxKubernetesSecurityPlatformAlert is a CEF custom key and alertMessageID is a syslog message ID
	alertMessageID    = "stackroxKubernetesSecurityPlatformAlert"
	auditLogMessageID = "stackroxKubernetesSecurityPlatformAudit"

	// Debug level
	testMessageSeverity = 7
	// Info level
	auditLogSeverity = 6
)

var (
	log = logging.LoggerForModule()

	// We could instead do abs(severity - 4) + 2 but I feel this is high maintenance and obfuscates the meaning
	alertToSyslogSeverityMap = map[storage.Severity]int{
		storage.Severity_UNSET_SEVERITY:    6, // informational
		storage.Severity_LOW_SEVERITY:      5, // notice
		storage.Severity_MEDIUM_SEVERITY:   4, // warning
		storage.Severity_HIGH_SEVERITY:     3, // error
		storage.Severity_CRITICAL_SEVERITY: 2, // critical
	}

	// CEF severities are 0-3=low, 4-6=medium, 7-8=high, 9-10=very high
	alertToCEFSeverityMap = map[storage.Severity]int{
		storage.Severity_UNSET_SEVERITY:    1, // low
		storage.Severity_LOW_SEVERITY:      3, // low
		storage.Severity_MEDIUM_SEVERITY:   5, // medium
		storage.Severity_HIGH_SEVERITY:     7, // high
		storage.Severity_CRITICAL_SEVERITY: 9, // very high
	}
)

func init() {
	notifiers.Add("syslog", func(notifier *storage.Notifier) (notifiers.Notifier, error) {
		return newSyslog(notifier)
	})
}

//go:generate mockgen-wrapper
type syslogSender interface {
	SendSyslog(syslogBytes []byte) error
	Cleanup()
}

type syslog struct {
	*storage.Notifier

	sender   syslogSender
	pid      int
	facility int
}

func validateSyslog(syslog *storage.Syslog) error {
	if syslog == nil {
		return errors.New("no syslog config found")
	}

	facility := syslog.GetLocalFacility()
	if facility < 0 || facility > 7 {
		return errors.Errorf("invalid facility %s must be between 0 and 7", facility.String())
	}

	return nil
}

func newSyslog(notifier *storage.Notifier) (*syslog, error) {
	if err := validateSyslog(notifier.GetSyslog()); err != nil {
		return nil, err
	}

	// This will have to account for local/UDP if we ever implement them
	sender, err := getTCPSender(notifier.GetSyslog().GetTcpConfig())
	if err != nil {
		return nil, err
	}

	pid := os.Getpid()

	facility := 8 * (int(notifier.GetSyslog().GetLocalFacility()) + 16)

	return &syslog{
		sender:   sender,
		Notifier: notifier,
		pid:      pid,
		facility: facility,
	}, nil
}

func auditLogToCEF(auditLog *v1.Audit_Message) string {
	extensionList := make([]string, 0, 8) // There will be 8 different key/value pairs in this message.

	// deviceReciptTime is allowed to be ms since epoch, seems easier than converting it to a time string
	extensionList = append(extensionList, makeTimestampExtensionPair(deviceReceiptTime, auditLog.GetTime())...)
	extensionList = append(extensionList, makeExtensionPair(sourceUserPrivileges, joinRoleNames(auditLog.GetUser().GetRoles())))
	extensionList = append(extensionList, makeExtensionPair(sourceUserName, auditLog.GetUser().GetUsername()))
	extensionList = append(extensionList, makeExtensionPair(requestURL, auditLog.GetRequest().GetEndpoint()))
	extensionList = append(extensionList, makeExtensionPair(requestMethod, auditLog.GetRequest().GetMethod()))
	extensionList = append(extensionList, makeExtensionPair(eventOutcome, auditLog.GetStatus().String()))
	extensionList = append(extensionList, makeExtensionPair(requestClientApplication, auditLog.GetMethod().String()))
	extensionList = append(extensionList, makeJSONExtensionPair(stackroxKubernetesSecurityPlatformAuditLog, auditLog))

	return getCEFHeaderWithExtension("AuditLog", "AuditLog", 3, makeExtensionFromPairs(extensionList))
}

func joinRoleNames(roles []*storage.UserInfo_Role) string {
	roleNames := make([]string, 0, len(roles))
	for _, r := range roles {
		roleNames = append(roleNames, r.GetName())
	}
	return strings.Join(roleNames, ",")
}

func alertToCEF(alert *storage.Alert) string {
	// There will be 4-5 different key/value pairs in this message.  Allocate space for 5 because additional
	// allocations are more expensive than a slightly large list
	extensionList := make([]string, 0, 5)

	extensionList = append(extensionList, makeExtensionPair(devicePayloadID, alert.GetId()))
	extensionList = append(extensionList, makeTimestampExtensionPair(startTime, alert.GetFirstOccurred())...)
	extensionList = append(extensionList, makeTimestampExtensionPair(deviceReceiptTime, alert.GetTime())...)
	if alert.GetState() == storage.ViolationState_RESOLVED {
		extensionList = append(extensionList, makeTimestampExtensionPair(endTime, alert.GetTime())...)
	}
	extensionList = append(extensionList, makeJSONExtensionPair(stackroxKubernetesSecurityPlatformAlert, alert))

	severity := alertToCEFSeverityMap[alert.GetPolicy().GetSeverity()]

	return getCEFHeaderWithExtension("Alert", alert.GetPolicy().GetName(), severity, makeExtensionFromPairs(extensionList))
}

func getCEFHeaderWithExtension(deviceEventClassID, name string, severity int, extension string) string {
	return fmt.Sprintf("CEF:0|StackRox|Kubernetes Security Platform|%s|%s|%d|%s|%s", version.GetMainVersion(), deviceEventClassID, severity, name, extension)
}

func makeExtensionPair(key, value string) string {
	return fmt.Sprintf("%s=%s", key, value)
}

func makeJSONExtensionPair(key string, valueObject interface{}) string {
	value, err := json.Marshal(valueObject)
	if err != nil {
		log.Warnf("unable to json marshal audit log field %s due to %v", key, err)
		return makeExtensionPair(key, "missing")
	}
	return makeExtensionPair(key, string(value))
}

func makeTimestampExtensionPair(key string, timestamp *types.Timestamp) []string {
	// string(seconds) + string(milliseconds) should result in the string representation of a millisecond timestamp
	if timestamp == nil {
		return nil
	}
	msts := strconv.Itoa(int((timestamp.Seconds)*1000 + int64(timestamp.Nanos/1000000)))
	return []string{makeExtensionPair(key, msts)}
}

func makeExtensionFromPairs(pairs []string) string {
	return strings.Join(pairs, " ")
}

func (s *syslog) wrapSyslogUnstructuredData(severity int, timestamp time.Time, messageID, unstructuredData string) string {
	priority := s.facility + severity

	return fmt.Sprintf("<%d>%d %s central %s %d %s - %s", priority, syslogVersion, timestamp.Format(time.RFC3339), application, s.pid, messageID, unstructuredData)
}

func (s *syslog) AlertNotify(ctx context.Context, alert *storage.Alert) error {
	unstructuredData := alertToCEF(alert)
	severity := alertToSyslogSeverityMap[alert.GetPolicy().GetSeverity()]
	timestamp, err := types.TimestampFromProto(alert.GetTime())
	if err != nil {
		return err
	}
	return s.sendSyslog(severity, timestamp, stackroxKubernetesSecurityPlatformAlert, unstructuredData)
}

func (s *syslog) Close(context.Context) error {
	s.sender.Cleanup()
	return nil
}

func (s *syslog) ProtoNotifier() *storage.Notifier {
	return s.Notifier
}

func (s *syslog) Test(context.Context) error {
	data := getCEFHeaderWithExtension("Test", "Test", 0, "stackroxKubernetesSecurityPlatformTestMessage=test")
	return s.sendSyslog(testMessageSeverity, time.Now(), "stackroxKubernetesSecurityPlatformIntegrationTest", data)
}

func (s *syslog) SendAuditMessage(ctx context.Context, msg *v1.Audit_Message) error {
	unstructuredData := auditLogToCEF(msg)
	timestamp, err := types.TimestampFromProto(msg.GetTime())
	if err != nil {
		return err
	}
	return s.sendSyslog(auditLogSeverity, timestamp, auditLogMessageID, unstructuredData)
}

func (s *syslog) AuditLoggingEnabled() bool {
	return true // TODO: Joseph this will have to change if we allow users to configure which messages are sent to splunk
}

func (s *syslog) sendSyslog(severity int, timestamp time.Time, messageID, unstructuredData string) error {
	syslog := s.wrapSyslogUnstructuredData(severity, timestamp, messageID, unstructuredData)
	return s.sender.SendSyslog([]byte(syslog))
}
