package codes

// Holds human-readable error codes for classifying errors emitted from logs.
// The error codes may be used for filtering within logs as well as rendering
// specific hints for the related administration events.
const (
	// API Token codes.
	TokenCreated = "token-created"
	TokenExpired = "token-expired"

	// Backup codes.
	GCSGeneric          = "gcs-generic"
	S3CompatibleGeneric = "s3compatible-generic"
	S3Generic           = "s3-generic"

	// Cloud Source codes.
	OCMCloudGeneric     = "ocm-generic"
	PaladinCloudGeneric = "paladin-cloud-generic"

	// Notifier codes.
	ACSCSEmailGeneric        = "acscs-email-generic"
	AWSSHBatchUpload         = "awssh-batch-upload"
	AWSSHCacheExhausted      = "awssh-cache-exhausted"
	AWSSHGeneric             = "awssh-generic"
	AWSSHHeartBeat           = "awssh-heartbeat"
	AWSSHInvalidTimestamp    = "awssh-timestamp"
	CloudPlatformGeneric     = "cloud-platform-generic"
	EmailGeneric             = "email-generic"
	JIRAGeneric              = "jira-generic"
	MicrosoftSentinelGeneric = "microsoft-sentinel-generic"
	PagerDutyGeneric         = "pagerduty-generic"
	SlackGeneric             = "slack-generic"
	SplunkGeneric            = "splunk-generic"
	SumoLogicGeneric         = "sumo-logic-generic"
	SyslogGeneric            = "syslog-generic"
	TeamsGeneric             = "teams-generic"
	WebhookGeneric           = "webhook-generic"
)
