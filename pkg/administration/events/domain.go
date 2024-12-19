package events

import "regexp"

const (
	authenticationDomain = "Authentication"
	defaultDomain        = "General"
	imageScanningDomain  = "Image Scanning"
	integrationDomain    = "Integrations"
)

var moduleToDomain = map[*regexp.Regexp]string{
	regexp.MustCompile(`^apitoken/expiration`):       authenticationDomain,
	regexp.MustCompile(`(^|/)externalbackups(/|$)`):  integrationDomain,
	regexp.MustCompile(`(^|/)cloudsources(/|$)`):     integrationDomain,
	regexp.MustCompile(`(^|/)notifiers(/|$)`):        integrationDomain,
	regexp.MustCompile(`^reprocessor|image/service`): imageScanningDomain,
}

// GetDomainFromModule retrieves a domain based on a specific module which will be
// used for administration events.
func GetDomainFromModule(module string) string {
	for regex, domain := range moduleToDomain {
		if regex.MatchString(module) {
			return domain
		}
	}
	return defaultDomain
}
