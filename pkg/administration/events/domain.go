package events

const (
	defaultDomain       = "General"
	imageScanningDomain = "Image Scanning"
	integrationDomain   = "Integrations"
)

var (
	// TODO(dhaus): Possibly switch to regexp for associating modules to domains.
	moduleToDomain = map[string]string{
		"reprocessor":   imageScanningDomain,
		"image/service": imageScanningDomain,
		// Notifiers.
		"pkg/notifiers/awssh":     integrationDomain,
		"pkg/notifiers/email":     integrationDomain,
		"pkg/notifiers/generic":   integrationDomain,
		"pkg/notifiers/jira":      integrationDomain,
		"pkg/notifiers/pagerduty": integrationDomain,
		"pkg/notifiers/slack":     integrationDomain,
		"pkg/notifiers/splunk":    integrationDomain,
		"pkg/notifiers/sumologic": integrationDomain,
		"pkg/notifiers/syslog":    integrationDomain,
		"pkg/notifiers/teams":     integrationDomain,
	}
)

// GetDomainFromModule retrieves a domain based on a specific module which will be
// used for administration events.
func GetDomainFromModule(module string) string {
	domain := moduleToDomain[module]
	if domain == "" {
		return defaultDomain
	}
	return domain
}
