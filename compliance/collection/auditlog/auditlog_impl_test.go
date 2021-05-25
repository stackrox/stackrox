package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/uuid"
	"github.com/stretchr/testify/suite"
)

//lint:file-ignore U1000 Unused functions are due to test skip.

type mockSender struct {
	sentC chan *auditEvent
}

func (c *mockSender) Send(ctx context.Context, event *auditEvent) error {
	go func() {
		c.sentC <- event
	}()
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

func (s *ComplianceAuditLogReaderTestSuite) TestReaderReturnsErrorIfFileExistsButCannotBeRead() {
	tempDir, err := ioutil.TempDir("", "")
	s.NoError(err)
	defer func() {
		s.NoError(os.RemoveAll(tempDir))
	}()
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

func (s *ComplianceAuditLogReaderTestSuite) TestReaderReturnsErrorIfSignalIsAlreadyDone() {
	logPath := "testdata/doesntmatter.log"
	_, reader := s.getMocks(logPath)

	reader.stopC.Signal()

	started, err := reader.StartReader(context.Background())
	s.False(started)
	s.Error(err, "It should fail with an error if signal is already stopped")
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderReturnsErrorIfReaderIsAlreadyStopped() {
	logPath := "testdata/doesntmatter.log"
	_, reader := s.getMocks(logPath)

	reader.StopReader()

	started, err := reader.StartReader(context.Background())
	s.False(started)
	s.Error(err, "It should fail with an error if reader is already stopped")
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderTailsLog() {
	s.T().Skipf("Temporarily skipping until deadlock is fixed in tests")
	tempDir, err := ioutil.TempDir("", "")
	s.NoError(err)
	defer func() {
		s.NoError(os.RemoveAll(tempDir))
	}()
	logPath := filepath.Join(tempDir, "testaudit_tail.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	line, expectedEvent := s.fakeAuditLogLine("get", "secrets", "fake-token", "stackrox")
	_, err = f.Write([]byte(line))
	s.NoError(err)

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event)

	// Write a few more log lines and check that they are read and parsed
	for i := 0; i < 5; i++ {
		line, expectedEvent = s.fakeAuditLogLine("get", "configmaps", fmt.Sprintf("fake-map%d", i), "stackrox")
		_, err = f.Write([]byte(line))
		s.NoError(err)

		event = s.getSentEvent(sender.sentC)
		s.Equal(expectedEvent, *event)
	}
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderOnlySendsEventsThatMatchFilter() {
	s.T().Skipf("Temporarily skipping until deadlock is fixed in tests")
	tempDir, err := ioutil.TempDir("", "")
	s.NoError(err)
	defer func() {
		s.NoError(os.RemoveAll(tempDir))
	}()
	logPath := filepath.Join(tempDir, "testaudit_filter.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a few log lines that should get filtered out
	for i := 0; i < 5; i++ {
		line, _ := s.fakeAuditLogLine("get", "something-else", "fake-thing", "stackrox")
		_, err = f.Write([]byte(line))
		s.NoError(err)
	}

	// Then something that doesn't get filtered out
	line, expectedEvent := s.fakeAuditLogLine("get", "secrets", "fake-token", "stackrox")
	_, err = f.Write([]byte(line))
	s.NoError(err)

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event) // First event received should match the one not filtered out
}

func (s *ComplianceAuditLogReaderTestSuite) TestReaderSkipsEventsThatCannotBeParsed() {
	s.T().Skipf("Temporarily skipping until deadlock is fixed in tests")
	tempDir, err := ioutil.TempDir("", "")
	s.NoError(err)
	defer func() {
		s.NoError(os.RemoveAll(tempDir))
	}()
	logPath := filepath.Join(tempDir, "testaudit_marshallerr.log")

	sender, reader := s.getMocks(logPath)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	s.NoError(err)
	defer s.cleanupFile(f, logPath)

	// Write a line that cannot be unmarshalled
	_, err = f.Write([]byte("This line cannot be unmarshalled!\n"))
	s.NoError(err)

	// And then something that won't get filtered out
	line, expectedEvent := s.fakeAuditLogLine("get", "secrets", "fake-token", "stackrox")
	_, err = f.Write([]byte(line))
	s.NoError(err)

	started, err := reader.StartReader(context.Background())
	s.True(started)
	s.NoError(err)
	defer reader.StopReader()

	event := s.getSentEvent(sender.sentC)
	s.Equal(expectedEvent, *event) // First event received should match the one not filtered out
}

func (s *ComplianceAuditLogReaderTestSuite) getMocks(logPath string) (*mockSender, *auditLogReaderImpl) {
	sender := &mockSender{
		sentC: make(chan *auditEvent, 5),
	}

	reader := &auditLogReaderImpl{
		logPath: logPath,
		stopC:   concurrency.NewSignal(),
		sender:  sender,
	}
	return sender, reader
}

func (s *ComplianceAuditLogReaderTestSuite) cleanupFile(f *os.File, path string) {
	_ = f.Close()
	err := os.Remove(path)
	s.NoError(err)
}

func (s *ComplianceAuditLogReaderTestSuite) fakeAuditLogLine(verb, resourceType, resourceName, namespace string) (string, auditEvent) {
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
		RequestReceivedTimestamp: "2021-05-06T00:19:49.906385Z",
		RequestURI:               uri,
		ResponseStatus: responseStatusRef{
			Metadata: nil,
			Status:   "",
			Message:  "",
			Code:     200,
		},
		SourceIPs:      []string{"10.0.119.155"},
		Stage:          "ResponseComplete",
		StageTimestamp: "2021-05-06T00:19:49.915375Z",
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
	afterCh := time.After(10 * time.Second)
	select {
	case event := <-c:
		return event
	case <-afterCh:
		s.FailNow("Channel didn't return after a while - might be a deadlock")
	}
	return nil // unreachable due to the FailNow but the compiler doesn't realize it hence the return
}
