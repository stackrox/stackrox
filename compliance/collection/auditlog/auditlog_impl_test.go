package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gogo/protobuf/types"
	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

type mockSender struct {
	sentC chan *auditEvent
}

func (c *mockSender) Send(_ context.Context, event *auditEvent) error {
	c.sentC <- event
	return nil
}

func TestComplianceAuditLogReader(t *testing.T) {
	suite.Run(t, new(ComplianceAuditLogReaderTestSuite))
}

type ComplianceAuditLogReaderTestSuite struct {
	suite.Suite
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderReturnsGracefullyIfFileDoesNotExist() {
	logPath := "testdata/does_not_exist.log"
	_, reader := s.getMocks(logPath)

	started, err := reader.StartReader(context.Background())
	s.False(started)
	s.NoError(err, "It shouldn't be an error if file doesn't exist")
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderStopDoesNotBlockIfStartFailed() {
	logPath := "testdata/does_not_exist.log"
	_, reader := s.getMocks(logPath)

	started, _ := reader.StartReader(context.Background())
	s.False(started)

	reader.StopReader()
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderReturnsErrorIfFileExistsButCannotBeRead() {
	// TODO(ROX-14204): enable this test on GHA
	if _, ok := os.LookupEnv("GITHUB_ACTIONS"); ok {
		s.T().Skip("ROX-14204: This test is not working on GHA.")
	}
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_notopenable.log")

	// Create a file that the reader cannot open due to permissions missing
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0000)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	_, reader := s.getMocks(logPath)

	started, err := reader.StartReader(context.Background())

	s.False(started)
	s.Error(err, "It should fail with an error if the log file is not openable")
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderTailsLog() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_tail.log")

	sender, reader := s.getMocks(logPath)

	eventTime := time.Now()
	line, expectedEvent := s.fakeAuditLogLineAtTime("get", "secrets", "fake-token", "stackrox", eventTime.Format(time.RFC3339Nano))
	s.writeToNewFile(logPath, line)

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event)

	// Write a few more log lines and check that they are read and parsed
	expectedEvents := make([]auditEvent, 0, 3)
	lines := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		eventTime = eventTime.Add(60 * time.Second)
		line, expectedEvent = s.fakeAuditLogLineAtTime("get", "configmaps", fmt.Sprintf("fake-map%d", i), "stackrox", eventTime.Format(time.RFC3339Nano))
		expectedEvents = append(expectedEvents, expectedEvent)
		lines = append(lines, line)
	}

	// Write to the file in parallel to simulate the log file getting written to in parallel
	go func() {
		for _, line := range lines {
			s.appendToFile(logPath, line)
		}
		// Write an extra line because the tailer and this test very rarely get stuck if the log just stops. This is an extremely
		// unlikely scenario is real life since the audit log is constantly being written to
		line, _ := s.fakeAuditLogLineAtTime("get", "configmaps", "extra-map", "stackrox", eventTime.Format(time.RFC3339Nano))
		s.appendToFile(logPath, line)
	}()

	time.Sleep(1 * time.Second)

	for _, expectedEvent := range expectedEvents {
		event = s.getSentEvent(sender.sentC)
		s.Equal(expectedEvent, *event)
	}
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderOnlySendsEventsThatMatchResourceTypeFilter() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_filter.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a few log lines that should get filtered out due to unsupported resource type
	for i := 0; i < 5; i++ {
		line, _ := s.fakeAuditLogLine("get", "something-else", "fake-thing", "stackrox")
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
	}

	// Then something that doesn't get filtered out
	validResources := map[string]string{
		"secrets":                    "fake-token",
		"configmaps":                 "fake-map",
		"clusterrolebindings":        "my-cluster-role-binding",
		"clusterroles":               "my-cluster-role",
		"networkpolicies":            "this-net-pol",
		"securitycontextconstraints": "s-c-c",
		"egressfirewalls":            "wall-that-fire",
	}
	var expectedEvents []auditEvent
	for r, n := range validResources {
		line, expectedEvent := s.fakeAuditLogLine("create", r, n, "stackrox")
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
		expectedEvents = append(expectedEvents, expectedEvent)
	}

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	// Give it a sec to catch up
	time.Sleep(1 * time.Second)

	for _, expectedEvent := range expectedEvents {
		event := s.getSentEvent(sender.sentC)
		s.Equal(expectedEvent, *event)
	}
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderOnlySendsEventsForValidStages() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_filter.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a few log lines for other stages
	unsupportedStages := []string{"RequestReceived", "ResponseStarted", "wut"}
	for _, stage := range unsupportedStages {
		line, _ := s.fakeAuditLogLineWithStage("get", "secrets", "fake-token", "stackrox", types.TimestampString(types.TimestampNow()), stage)
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
	}

	// Then ResponseComplete and Panic as those are allowed
	supportedStages := []string{"ResponseComplete", "Panic"}
	expectedEvents := make([]auditEvent, 0, 2)
	for _, stage := range supportedStages {
		line, expectedEvent := s.fakeAuditLogLineWithStage("get", "secrets", "fake-token", "stackrox", types.TimestampString(types.TimestampNow()), stage)
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
		expectedEvents = append(expectedEvents, expectedEvent)
	}

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	// Give it a sec to catch up
	time.Sleep(1 * time.Second)

	for _, expectedEvent := range expectedEvents {
		event := s.getSentEvent(sender.sentC)
		s.Equal(expectedEvent, *event) // First event received should match the one not filtered out
	}
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderOnlySendsEventsForVerbsNotInDenyList() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_filter_verb.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a few log lines for verbs that won't be sent
	unsupportedVerbs := []string{"WATCH", "watch", "LIST", "list", "Watch", "lIsT"}
	for _, verb := range unsupportedVerbs {
		line, _ := s.fakeAuditLogLineWithStage(verb, "secrets", "fake-token", "stackrox", types.TimestampString(types.TimestampNow()), "ResponseComplete")
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
	}

	// Then the ones that will be
	supportedVerbs := []string{"GET", "get", "Create", "patCH", "DELETE"}
	expectedEvents := make([]auditEvent, 0, 2)
	for _, verb := range supportedVerbs {
		line, expectedEvent := s.fakeAuditLogLineWithStage(verb, "secrets", "fake-token", "stackrox", types.TimestampString(types.TimestampNow()), "ResponseComplete")
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
		expectedEvents = append(expectedEvents, expectedEvent)
	}

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	// Give it a sec to catch up
	time.Sleep(1 * time.Second)

	for _, expectedEvent := range expectedEvents {
		event := s.getSentEvent(sender.sentC)
		s.Equal(expectedEvent, *event) // First event received should match the one not filtered out
	}
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderSkipsEventsThatCannotBeParsed() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_marshallerr.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a line that cannot be unmarshalled
	_, err = f.Write([]byte("This line cannot be unmarshalled!\n"))
	s.NoError(err)
	s.NoError(f.Sync())

	// And then something that won't get filtered out
	line, expectedEvent := s.fakeAuditLogLine("get", "secrets", "fake-token", "stackrox")
	_, err = f.Write([]byte(line))
	s.NoError(err)
	s.NoError(f.Sync())

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event) // First event received should match the one not filtered out

	reader.StopReader() // force stop
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderStartsSendingEventsAfterStartState() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_filestate.log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a few log lines that are in the past
	eventTime := time.Now()
	var latestEvent auditEvent
	var line string
	for i := 0; i < 5; i++ {
		eventTime = eventTime.Add(1 * time.Second)
		line, latestEvent = s.fakeAuditLogLineAtTime("get", "secrets", "fake-token", "stackrox", eventTime.Format(time.RFC3339Nano))
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
	}

	// Then something that that's after `CollectLogsSince` which shouldn't be filtered out
	line, expectedEvent := s.fakeAuditLogLineAtTime("get", "secrets", "new-fake-token", "stackrox", eventTime.Add(1*time.Minute).Format(time.RFC3339Nano))
	_, err = f.Write([]byte(line))
	s.NoError(err)
	s.NoError(f.Sync())

	collectSinceTs, _ := types.TimestampProto(eventTime)
	sender, reader := s.getMocksWithStartState(logPath, &storage.AuditLogFileState{
		CollectLogsSince: collectSinceTs,
		LastAuditId:      latestEvent.AuditID,
	})

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event) // First event received should match the one after CollectLogsSince
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderStartsSendingEventsAtStartStateIfIdsDontMatch() {
	tempDir := s.T().TempDir()
	logPath := filepath.Join(tempDir, "testaudit_filestate.log")

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a few lines to start
	eventTime := time.Now().Add(1 * time.Minute)
	var latestEvent auditEvent
	var line string
	for i := 0; i < 5; i++ {
		eventTime = eventTime.Add(1 * time.Second)
		line, latestEvent = s.fakeAuditLogLineAtTime("get", "secrets", "fake-token", "stackrox", eventTime.Format(time.RFC3339Nano))
		_, err = f.Write([]byte(line))
		s.NoError(err)
		s.NoError(f.Sync())
	}

	// Write a new line at the exact same time as the last one written
	line, expectedEvent := s.fakeAuditLogLineAtTime("get", "secrets", "new-fake-token", "stackrox", eventTime.Format(time.RFC3339Nano))
	_, err = f.Write([]byte(line))
	s.NoError(err)
	s.NoError(f.Sync())

	// Set CollectLogSince to be same exact time as the last two logs, but the ID should be the first of the two
	collectSinceTs, _ := types.TimestampProto(eventTime)
	sender, reader := s.getMocksWithStartState(logPath, &storage.AuditLogFileState{
		CollectLogsSince: collectSinceTs,
		LastAuditId:      latestEvent.AuditID,
	})

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event) // First event received should match the last one written
}

func (s *ComplianceAuditLogReaderTestSuite) getMocks(logPath string) (*mockSender, *auditLogReaderImpl) {
	sender := &mockSender{
		sentC: make(chan *auditEvent, 10), // large enough to be able to buffer everything in the tests
	}

	reader := &auditLogReaderImpl{
		logPath: logPath,
		stopper: concurrency.NewStopper(),
		sender:  sender,
	}
	return sender, reader
}

func (s *ComplianceAuditLogReaderTestSuite) writeToNewFile(logPath string, lines ...string) {
	err := os.WriteFile(logPath, []byte(strings.Join(lines, "\n")), 0600)
	s.NoError(err)
}

func (s *ComplianceAuditLogReaderTestSuite) appendToFile(logPath string, lines ...string) {
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)
	s.NoError(err)

	defer func() {
		s.NoError(f.Close())
	}()

	_, err = f.WriteString(strings.Join(lines, "\n"))
	s.NoError(err)
}

func (s *ComplianceAuditLogReaderTestSuite) getMocksWithStartState(logPath string, startState *storage.AuditLogFileState) (*mockSender, *auditLogReaderImpl) {
	sender := &mockSender{
		sentC: make(chan *auditEvent, 5),
	}

	reader := &auditLogReaderImpl{
		logPath:    logPath,
		stopper:    concurrency.NewStopper(),
		sender:     sender,
		startState: startState,
	}
	return sender, reader
}

func (s *ComplianceAuditLogReaderTestSuite) cleanupFile(f *os.File, path string) {
	_ = f.Close()
	err := os.Remove(path)
	s.NoError(err)
}

func (s *ComplianceAuditLogReaderTestSuite) fakeAuditLogLine(verb, resourceType, resourceName, namespace string) (string, auditEvent) {
	return s.fakeAuditLogLineAtTime(verb, resourceType, resourceName, namespace, "2021-05-06T00:19:49.915375Z")
}

func (s *ComplianceAuditLogReaderTestSuite) fakeAuditLogLineAtTime(verb, resourceType, resourceName, namespace, time string) (string, auditEvent) {
	return s.fakeAuditLogLineWithStage(verb, resourceType, resourceName, namespace, time, "ResponseComplete")
}

func (s *ComplianceAuditLogReaderTestSuite) fakeAuditLogLineWithStage(verb, resourceType, resourceName, namespace, time, stage string) (string, auditEvent) {
	uri := fmt.Sprintf("/api/v1/namespaces/stackrox/%s/%s", resourceType, resourceName)
	event := auditEvent{
		Annotations: map[string]string{
			"authorization.k8s.io/decision": "allow",
			"authorization.k8s.io/reason":   "RBAC: allowed by RoleBinding \"stackrox-central-diagnostics/stackrox\" of Role \"stackrox-central-diagnostics\" to ServiceAccount \"central/stackrox\"",
		},
		APIVersion: "audit.k8s.io/v1",
		AuditID:    uuid.NewV4().String(),
		Kind:       "Event",
		Level:      "Metadata",
		ObjectRef: objectRef{
			APIVersion: "v1",
			Name:       resourceName,
			Namespace:  namespace,
			Resource:   resourceType,
		},
		RequestReceivedTimestamp: time,
		RequestURI:               uri,
		ResponseStatus: responseStatusRef{
			Metadata: nil,
			Status:   "",
			Message:  "",
			Code:     200,
		},
		SourceIPs:      []string{"10.0.119.155"},
		Stage:          stage,
		StageTimestamp: time,
		User: userRef{
			Username: "cluster-admin",
			UID:      "56d060c4-363a-4d1f-bffc-b146078ccb8e",
			Groups:   []string{"cluster-admins", "system:authenticated:oauth", "system:authenticated"},
		},
		ImpersonatedUser: &userRef{
			Username: "system:serviceaccount:stackrox:central",
			UID:      "",
			Groups:   []string{"system:serviceaccounts", "system:serviceaccounts:stackrox", "system:authenticated"},
		},
		UserAgent: "oc/4.7.0 (darwin/amd64) kubernetes/c66c03f",
		Verb:      verb,
	}

	line, err := json.Marshal(event)
	s.NoError(err)
	return fmt.Sprintf("%s\n", line), event
}

func (s *ComplianceAuditLogReaderTestSuite) getSentEvent(c chan *auditEvent) *auditEvent {
	afterCh := time.After(20 * time.Second)
	select {
	case event := <-c:
		return event
	case <-afterCh:
		s.FailNow("Channel didn't return after a while - might be a deadlock")
	}
	return nil // unreachable due to the FailNow but the compiler doesn't realize it hence the return
}
