package centralevents

const (
	defaultDomain       = "General"
	imageScanningDomain = "Image Scanning"
)

var (
	moduleToDomain = map[string]string{
		"reprocessor":   imageScanningDomain,
		"image/service": imageScanningDomain,
	}
)

// GetDomainFromModule retrieves a domain based on a specific module which will be
// used for Central events.
func GetDomainFromModule(module string) string {
	domain := moduleToDomain[module]
	if domain == "" {
		return defaultDomain
	}
	return domain
}
