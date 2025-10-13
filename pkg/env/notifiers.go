package env

import "time"

var (
	// AWSSHUploadTimeout specifies the upload timeout for the AWS Security Hub notifier.
	AWSSHUploadTimeout = registerDurationSetting("ROX_AWSSH_UPLOAD_TIMEOUT", 2*time.Second)
	// AWSSHUploadInterval specifies the interval for uploading alerts to AWS Security Hub.
	AWSSHUploadInterval = registerDurationSetting("ROX_AWSSH_UPLOAD_INTERVAL", 15*time.Second)
	// SyslogUploadTimeout specifies the upload timeout for the Syslog notifier.
	SyslogUploadTimeout = registerDurationSetting("ROX_SYSLOG_UPLOAD_TIMEOUT", 5*time.Second)
	// TeamsTimeout specifies the timeout for posting messages to Teams via the Teams notifier.
	TeamsTimeout = registerDurationSetting("ROX_TEAMS_TIMEOUT", 10*time.Second)
	// CSCCTimeout specifies the timeout for sending alerts to Google Cloud Security Platform.
	CSCCTimeout = registerDurationSetting("ROX_CSCC_TIMEOUT", 5*time.Second)
	// WebhookTimeout specifies the timeout for sending alerts via the generic webhook notifier.
	WebhookTimeout = registerDurationSetting("ROX_WEBHOOK_TIMEOUT", 5*time.Second)
)
