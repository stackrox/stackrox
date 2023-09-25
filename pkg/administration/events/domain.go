package events

import "regexp"

const (
	defaultDomain       = "General"
	imageScanningDomain = "Image Scanning"
	integrationDomain   = "Integrations"
)

var (
	moduleToDomain = map[*regexp.Regexp]string{
		regexp.MustCompile(`^reprocessor|image/service`): imageScanningDomain,
		regexp.MustCompile(`^pkg/notifiers(/|$)`):        integrationDomain,
	}
)

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
