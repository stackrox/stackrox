package external

import (
	"fmt"
	"time"

	testenv "github.com/stackrox/rox/pkg/testutils/env"
)

type NotificationType string

const (
	SlackNotification   NotificationType = "slack"
	EmailNotification   NotificationType = "email"
	WebhookNotification NotificationType = "webhook"
	SplunkNotification  NotificationType = "splunk"
	MockNotification    NotificationType = "mock"
)

// NotificationMessage represents a message to be sent via notification service
type NotificationMessage struct {
	Title      string
	Text       string
	Channel    string
	Recipients []string
	Severity   string
	Timestamp  time.Time
	Metadata   map[string]string
}

// NotificationResponse represents the response from a notification service
type NotificationResponse struct {
	MessageID string
	Status    string
	Timestamp time.Time
	Error     string
}

// Slack Client
type SlackClient struct {
	webhookURL string
	mock       bool
	sentMessages []*NotificationMessage
}

func NewSlackClient() (*SlackClient, error) {
	webhookURL := testenv.SlackWebhookURL.Setting()

	// Use mock if no credentials or in development mode
	if webhookURL == "" || testenv.ShouldUseMockServices() {
		return &SlackClient{mock: true, sentMessages: make([]*NotificationMessage, 0)}, nil
	}

	return &SlackClient{webhookURL: webhookURL, mock: false}, nil
}

func (s *SlackClient) SendMessage(message *NotificationMessage) error {
	if s.mock {
		message.Timestamp = time.Now()
		s.sentMessages = append(s.sentMessages, message)
		if message.Text == "fail" {
			return fmt.Errorf("mock notification failure")
		}
		return nil
	}

	// TODO: Implement real Slack webhook message sending
	// This would use HTTP POST to the webhook URL with proper Slack message format
	return fmt.Errorf("Slack message sending not implemented")
}

func (s *SlackClient) TestConnection() error {
	if s.mock {
		return nil
	}

	// TODO: Implement real Slack connection test
	return fmt.Errorf("Slack connection test not implemented")
}

func (s *SlackClient) GetNotificationType() NotificationType {
	return SlackNotification
}

// GetSentMessages returns all messages sent through this mock client (for test verification)
func (s *SlackClient) GetSentMessages() []*NotificationMessage {
	if s.mock {
		return s.sentMessages
	}
	return nil
}

// ClearSentMessages clears the sent messages history
func (s *SlackClient) ClearSentMessages() {
	if s.mock {
		s.sentMessages = make([]*NotificationMessage, 0)
	}
}

// Email Client
type EmailClient struct {
	smtpHost    string
	smtpPort    int
	username    string
	password    string
	fromAddress string
	mock        bool
	sentMessages []*NotificationMessage
}

func NewEmailClient() (*EmailClient, error) {
	// Email credentials would need to be added to testenv
	// For now, always use mock
	if testenv.ShouldUseMockServices() {
		return &EmailClient{mock: true, sentMessages: make([]*NotificationMessage, 0)}, nil
	}

	return nil, fmt.Errorf("email notification not implemented")
}

func (e *EmailClient) SendMessage(message *NotificationMessage) error {
	if e.mock {
		message.Timestamp = time.Now()
		e.sentMessages = append(e.sentMessages, message)
		if message.Text == "fail" {
			return fmt.Errorf("mock notification failure")
		}
		return nil
	}

	// TODO: Implement real SMTP email sending
	return fmt.Errorf("email sending not implemented")
}

func (e *EmailClient) TestConnection() error {
	if e.mock {
		return nil
	}

	// TODO: Implement real SMTP connection test
	return fmt.Errorf("email connection test not implemented")
}

func (e *EmailClient) GetNotificationType() NotificationType {
	return EmailNotification
}

// Webhook Client
type WebhookClient struct {
	serverCA string
	endpoint string
	mock     bool
	sentMessages []*NotificationMessage
}

func NewWebhookClient() (*WebhookClient, error) {
	serverCA := testenv.GenericWebhookServerCA.Setting()

	if serverCA == "" || testenv.ShouldUseMockServices() {
		return &WebhookClient{mock: true, sentMessages: make([]*NotificationMessage, 0)}, nil
	}

	return &WebhookClient{serverCA: serverCA, mock: false}, nil
}

func (w *WebhookClient) SendMessage(message *NotificationMessage) error {
	if w.mock {
		message.Timestamp = time.Now()
		w.sentMessages = append(w.sentMessages, message)
		if message.Text == "fail" {
			return fmt.Errorf("mock notification failure")
		}
		return nil
	}

	// TODO: Implement real generic webhook posting
	return fmt.Errorf("webhook message sending not implemented")
}

func (w *WebhookClient) TestConnection() error {
	if w.mock {
		return nil
	}

	// TODO: Implement real webhook connection test
	return fmt.Errorf("webhook connection test not implemented")
}

func (w *WebhookClient) GetNotificationType() NotificationType {
	return WebhookNotification
}

// Splunk Client
type SplunkClient struct {
	endpoint string
	token    string
	index    string
	mock     bool
	sentMessages []*NotificationMessage
}

func NewSplunkClient() (*SplunkClient, error) {
	// Splunk credentials would need to be added to testenv
	// For now, always use mock
	if testenv.ShouldUseMockServices() {
		return &SplunkClient{mock: true, sentMessages: make([]*NotificationMessage, 0)}, nil
	}

	return nil, fmt.Errorf("Splunk notification not implemented")
}

func (s *SplunkClient) SendMessage(message *NotificationMessage) error {
	if s.mock {
		message.Timestamp = time.Now()
		s.sentMessages = append(s.sentMessages, message)
		if message.Text == "fail" {
			return fmt.Errorf("mock notification failure")
		}
		return nil
	}

	// TODO: Implement real Splunk HEC event forwarding
	return fmt.Errorf("Splunk message sending not implemented")
}

func (s *SplunkClient) TestConnection() error {
	if s.mock {
		return nil
	}

	// TODO: Implement real Splunk connection test
	return fmt.Errorf("Splunk connection test not implemented")
}

func (s *SplunkClient) GetNotificationType() NotificationType {
	return SplunkNotification
}

// Helper functions for creating test messages

// NewTestMessage creates a test notification message
func NewTestMessage(title, text string) *NotificationMessage {
	return &NotificationMessage{
		Title:     title,
		Text:      text,
		Channel:   "#test-channel",
		Severity:  "INFO",
		Timestamp: time.Now(),
		Metadata:  make(map[string]string),
	}
}

// NewAlertMessage creates a notification message for a policy alert
func NewAlertMessage(policyName, deploymentName, severity string) *NotificationMessage {
	return &NotificationMessage{
		Title:     fmt.Sprintf("Policy Violation: %s", policyName),
		Text:      fmt.Sprintf("Deployment %s violated policy %s", deploymentName, policyName),
		Channel:   "#security-alerts",
		Severity:  severity,
		Timestamp: time.Now(),
		Metadata: map[string]string{
			"policy":     policyName,
			"deployment": deploymentName,
			"type":       "policy_violation",
		},
	}
}
