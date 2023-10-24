package codes

// Holds human-readable error codes for classifying errors emitted from logs.
// The error codes may be used for filtering within logs as well as rendering
// specific hints for the related administration events.
const (
	AWSSHGeneric          = "awssh-generic"
	AWSSHHeartBeat        = "awssh-heartbeat"
	AWSSHInvalidTimestamp = "awssh-timestamp"
	AWSSHBatchUpload      = "awssh-batch-upload"
	AWSSHCacheExhausted   = "awssh-cache-exhausted"

	CloudPlatformGeneric = "cloud-platform-generic"

	WebhookGeneric = "webhook-generic"

	EmailGeneric = "email-generic"

	JIRAGeneric = "jira-generic"

	PagerDutyGeneric = "pagerduty-generic"

	SlackGeneric = "slack-generic"

	SplunkGeneric = "splunk-generic"

	SumoLogicGeneric = "sumo-logic-generic"

	SyslogGeneric = "syslog-generic"

	TeamsGeneric = "teams-generic"
)
