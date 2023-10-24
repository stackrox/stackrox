package events

import (
	"github.com/stackrox/rox/pkg/administration/events/codes"
	adminResources "github.com/stackrox/rox/pkg/administration/events/resources"
)

const (
	defaultRemediation = "An unknown issue occurred. Make sure to check out the detailed event message for more details."
)

var (
	// Currently, a hint is based on the domain and the resource associated with an administration event.
	// Additionally, an optional error code can be given based on the resource to give a more specialized hint.
	// In the future, we may extend this, and possibly also ensure hints are loaded externally (similar to
	// vulnerability definitions).
	hints = map[string]map[string]map[string]string{
		authenticationDomain: {
			adminResources.APIToken: {
				"": `An API token is about to expire. See the details on the expiration time within the event message.
You cannot re-create the token. Instead, perform these steps:
- Delete the expiring API token.
- Create a new API token (you can choose the same name).

You can then use the newly-created API token.
`,
			},
		},
		defaultDomain: {},
		imageScanningDomain: {
			// For now, this is an example string. We may want to revisit those together with UX / the docs team to get
			// errors that are in-line with documentation guidelines.
			adminResources.Image: {
				"": `An issue occurred scanning the image. Ensure that:
- Scanner can access the registry.
- Correct credentials are configured for the particular registry or repository.
- The scanned manifest exists within the registry or repository.`,
			},
		},
		integrationDomain: {
			adminResources.Notifier: {
				codes.AWSSHGeneric: `An issue occurred when using the AWS Security Hub notifier.
Ensure that:
- Credentials are configured correctly.
- Central can access AWS Security Hub.

For additional information, consult the event message for detailed information.`,
				codes.AWSSHHeartBeat: `Heartbeat messages to AWS Security Hub cannot be sent.
This indicates that the integration is not working as expected. Ensure that:
- Credentials are configured correctly.
- Central can access AWS Security Hub.`,
				codes.AWSSHInvalidTimestamp: `An incoming alert could not be sent to AWS Security Hub due to an invalid timestamp.
Verify the referenced alert for correctness.

If the issue persists, open a ticket with RHACS support.`,
				codes.AWSSHBatchUpload: `Uploading alerts to the AWS Security hub failed.
This leads to AWS Security Hub potentially missing some or all alerts emitted from Central.
Ensure that:
- Credentials are configured correctly.
- Central can access AWS Security Hub.

In case a timeout happened, adjust the timeout for uploading alerts to AWS Security Hub by setting the "ROX_AWSSH_UPLOAD_TIMEOUT"
environment variable.`,
				codes.AWSSHCacheExhausted: `The cache of alerts to-be-uploaded to AWS Security Hub is increasing.
This will lead to an increasing delay in alerts being uploaded to AWS Security Hub.

Adjust the upload interval for uploading alerts to AWS Security Hub by setting the "ROX_AWSSH_UPLOAD_INTERVAL"
environment variable.`,
				codes.EmailGeneric: `An issue occurred when using the Email notifier.
Ensure that:
- Configuration is valid, specifically the auth information and TLS configuration.
- Central can access the SMTP endpoint.

For additional information, consult the event message for detailed information.`,
				codes.JIRAGeneric: `An issue occurred when creating an issue via the JIRA notifier.
Ensure that:
- Configuration is valid, specifically the auth information.
- Central can access the JIRA endpoint.

For additional information, consult the event message for detailed information.`,
				codes.PagerDutyGeneric: `An issue occurred when using the PagerDuty notifier.
Ensure that:
- Configuration is valid, specifically the auth information.
- Central can access the PagerDuty endpoint.

For additional information, consult the event message for detailed information.`,
				codes.SlackGeneric: `An issue occurred when using the Slack notifier.
Ensure that:
- Configuration is valid, specifically the auth information.
- Central can access the Slack endpoint.

For additional information, consult the event message for detailed information.`,
				codes.SyslogGeneric: `An issue occurred when using the Syslog notifier.
Ensure that:
- The message format is valid.
- Central can access the Syslog endpoint.

In case a timeout error occurred, adjust the timeout for sending alerts to Syslog by setting the "ROX_SYSLOG_TIMEOUT"
environment variable, and increase the default timeout of 5 seconds.`,
				codes.SplunkGeneric: `An issue occurred when using the Splunk notifier.
Ensure that:
- Configuration is valid, specifically the Splunk HEC's vadility.
- Central can access the Splunk endpoint.

For additional information, consult the event message for detailed information.`,
				codes.SumoLogicGeneric: `An issue occurred when using the Sumo Logic notifier.
Ensure that:
- Configuration is valid, specifically the TLS configuration.
- Central can access the Sumo logic endpoint.

For additional information, consult the event message for detailed information.`,
				codes.TeamsGeneric: `An issue occurred when using the Teams notifier.
Ensure that:
- Configuration is valid.
- Central can access the Teams endpoint.

In case a timeout error occurred, adjust the timeout for sending alerts to teams by setting the "ROX_TEAMS_TIMEOUT"
environment variable.`,
				codes.CloudPlatformGeneric: `An issue occurred when using the Cloud Security Platform notifier.
Ensure that:
- Configuration is valid.
- Central can access the Cloud Security platform endpoint.

In case a timeout error occurred, adjust the timeout for sending alerts by setting the "ROX_CSCC_TIMEOUT"
environment variable.`,
				codes.WebhookGeneric: `An issue occurred when using the Generic notifier.
Ensure that:
- Configuration is valid, specifically the auth information and TLS certificates.
- Central can access the webhook endpoint.

In case a timeout error occurred, adjust the timeout for sending alerts by setting the "ROX_WEBHOOK_TIMEOUT"
environment variable.`,
			},
		},
	}
)

// GetHint retrieves the hint for a specific domain and resource.
// In case no hint exists for a specific area and resource, a generic text will be added.
//
// Currently, each hint is hardcoded and cannot be updated. In the future
// it might be possible to fetch definitions for a hint externally (e.g. similar to vulnerability definitions).
func GetHint(domain string, resource string, errCode string) string {
	hintPerResource := hints[domain]
	if hintPerResource == nil {
		return defaultRemediation
	}

	hints := hintPerResource[resource]
	if hints == nil {
		return defaultRemediation
	}

	return hints[errCode]
}
