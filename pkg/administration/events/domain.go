package events

import "regexp"

const (
	AuthenticationDomain = "Authentication"
	DefaultDomain        = "General"
	ImageScanningDomain  = "Image Scanning"
	IntegrationDomain    = "Integrations"
)

var moduleToDomain = map[*regexp.Regexp]string{
	regexp.MustCompile(`^apitoken/expiration`):       AuthenticationDomain,
	regexp.MustCompile(`(^|/)externalbackups(/|$)`):  IntegrationDomain,
	regexp.MustCompile(`(^|/)cloudsources(/|$)`):     IntegrationDomain,
	regexp.MustCompile(`(^|/)notifiers(/|$)`):        IntegrationDomain,
	regexp.MustCompile(`^reprocessor|image/service`): ImageScanningDomain,
}

// GetDomainFromModule retrieves a domain based on a specific module which will be
// used for administration events.
func GetDomainFromModule(module string) string {
	for regex, domain := range moduleToDomain {
		if regex.MatchString(module) {
			return domain
		}
	}
	return DefaultDomain
}
