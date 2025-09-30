package external

import (
	"fmt"
	"time"

	"github.com/stackrox/rox/pkg/testutils/credentials"
)

// NotificationClient interface for notification service operations
type NotificationClient interface {
	SendMessage(message *NotificationMessage) error
	TestConnection() error
	GetNotificationType() NotificationType
}

type NotificationType string

const (
	SlackNotification    NotificationType = "slack"
	EmailNotification    NotificationType = "email"
	WebhookNotification  NotificationType = "webhook"
	SplunkNotification   NotificationType = "splunk"
	MockNotification     NotificationType = "mock"
)

// NotificationMessage represents a message to be sent via notification service
type NotificationMessage struct {
	Title       string
	Text        string
	Channel     string
	Recipients  []string
	Severity    string
	Timestamp   time.Time
	Metadata    map[string]string
}

// NotificationResponse represents the response from a notification service
type NotificationResponse struct {
	MessageID string
	Status    string
	Timestamp time.Time
	Error     string
}

// NewNotificationClient creates a notification client based on available credentials
func NewNotificationClient(creds *credentials.Credentials, notificationType NotificationType) (NotificationClient, error) {
	// Check if we should use mocks
	if creds.ShouldUseMockServices() {
		return NewMockNotificationClient(notificationType), nil
	}

	// Create real client based on type and available credentials
	switch notificationType {
	case SlackNotification:
		if !creds.HasSlackCredentials() {
			if creds.IsDevelopmentMode() {
				return NewMockNotificationClient(SlackNotification), nil
			}
			return nil, fmt.Errorf("Slack credentials required")
		}
		return NewSlackClient(creds.SlackWebhookURL)

	case EmailNotification:
		// Email credentials would need to be added to credentials.go
		if creds.IsDevelopmentMode() {
			return NewMockNotificationClient(EmailNotification), nil
		}
		return nil, fmt.Errorf("email notification not implemented")

	case WebhookNotification:
		if creds.GenericWebhookServerCA == "" {
			if creds.IsDevelopmentMode() {
				return NewMockNotificationClient(WebhookNotification), nil
			}
			return nil, fmt.Errorf("webhook credentials required")
		}
		return NewWebhookClient(creds.GenericWebhookServerCA)

	case SplunkNotification:
		// Splunk credentials would need to be added
		if creds.IsDevelopmentMode() {
			return NewMockNotificationClient(SplunkNotification), nil
		}
		return nil, fmt.Errorf("Splunk notification not implemented")

	default:
		return nil, fmt.Errorf("unsupported notification type: %s", notificationType)
	}
}

// GetAvailableNotificationClients returns notification clients that can be created with current credentials
func GetAvailableNotificationClients(creds *credentials.Credentials) []NotificationClient {
	var clients []NotificationClient

	notificationTypes := []NotificationType{
		SlackNotification, EmailNotification, WebhookNotification, SplunkNotification,
	}

	for _, notifType := range notificationTypes {
		client, err := NewNotificationClient(creds, notifType)
		if err == nil {
			clients = append(clients, client)
		}
	}

	return clients
}

// Mock Notification Client Implementation
type MockNotificationClient struct {
	notificationType NotificationType
	sentMessages     []*NotificationMessage
}

func NewMockNotificationClient(notificationType NotificationType) *MockNotificationClient {
	return &MockNotificationClient{
		notificationType: notificationType,
		sentMessages:     make([]*NotificationMessage, 0),
	}
}

func (m *MockNotificationClient) SendMessage(message *NotificationMessage) error {
	// Store message for verification in tests
	message.Timestamp = time.Now()
	m.sentMessages = append(m.sentMessages, message)

	// Simulate different responses based on message content
	if message.Text == "fail" {
		return fmt.Errorf("mock notification failure")
	}

	return nil
}

func (m *MockNotificationClient) TestConnection() error {
	// Mock connection test always succeeds
	return nil
}

func (m *MockNotificationClient) GetNotificationType() NotificationType {
	return m.notificationType
}

// GetSentMessages returns all messages sent through this mock client (for test verification)
func (m *MockNotificationClient) GetSentMessages() []*NotificationMessage {
	return m.sentMessages
}

// ClearSentMessages clears the sent messages history
func (m *MockNotificationClient) ClearSentMessages() {
	m.sentMessages = make([]*NotificationMessage, 0)
}

// Real notification client implementations (stubs)

// Slack Client
type SlackClient struct {
	webhookURL string
}

func NewSlackClient(webhookURL string) (*SlackClient, error) {
	return &SlackClient{webhookURL: webhookURL}, nil
}

func (s *SlackClient) SendMessage(message *NotificationMessage) error {
	// TODO: Implement real Slack webhook message sending
	// This would use HTTP POST to the webhook URL with proper Slack message format
	return fmt.Errorf("Slack message sending not implemented")
}

func (s *SlackClient) TestConnection() error {
	// TODO: Implement real Slack connection test
	return fmt.Errorf("Slack connection test not implemented")
}

func (s *SlackClient) GetNotificationType() NotificationType {
	return SlackNotification
}

// Email Client
type EmailClient struct {
	smtpHost     string
	smtpPort     int
	username     string
	password     string
	fromAddress  string
}

func NewEmailClient(smtpHost string, smtpPort int, username, password, fromAddress string) (*EmailClient, error) {
	return &EmailClient{
		smtpHost:    smtpHost,
		smtpPort:    smtpPort,
		username:    username,
		password:    password,
		fromAddress: fromAddress,
	}, nil
}

func (e *EmailClient) SendMessage(message *NotificationMessage) error {
	// TODO: Implement real SMTP email sending
	return fmt.Errorf("email sending not implemented")
}

func (e *EmailClient) TestConnection() error {
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
}

func NewWebhookClient(serverCA string) (*WebhookClient, error) {
	return &WebhookClient{serverCA: serverCA}, nil
}

func (w *WebhookClient) SendMessage(message *NotificationMessage) error {
	// TODO: Implement real generic webhook posting
	return fmt.Errorf("webhook message sending not implemented")
}

func (w *WebhookClient) TestConnection() error {
	// TODO: Implement real webhook connection test
	return fmt.Errorf("webhook connection test not implemented")
}

func (w *WebhookClient) GetNotificationType() NotificationType {
	return WebhookNotification
}

// Splunk Client
type SplunkClient struct {
	endpoint   string
	token      string
	index      string
}

func NewSplunkClient(endpoint, token, index string) (*SplunkClient, error) {
	return &SplunkClient{
		endpoint: endpoint,
		token:    token,
		index:    index,
	}, nil
}

func (s *SplunkClient) SendMessage(message *NotificationMessage) error {
	// TODO: Implement real Splunk HEC event forwarding
	return fmt.Errorf("Splunk message sending not implemented")
}

func (s *SplunkClient) TestConnection() error {
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