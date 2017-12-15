package notifiers

import (
	"log"
	"net/url"

	"bitbucket.org/stack-rox/apollo/pkg/api/generated/api/v1"
)

const alertLinkPath = "/violations"

// AlertLink is the link URL for this alert
func AlertLink(alert *v1.Alert, endpoint string) string {
	base, err := url.Parse(endpoint)
	if err != nil {
		log.Print(err)
	}
	u, err := url.Parse(alertLinkPath)
	if err != nil {
		log.Print(err)
	}
	return base.ResolveReference(u).String()
}

// SeverityString is the nice form of the Severity enum
func SeverityString(s v1.Severity) string {
	switch s {
	case v1.Severity_UNSET_SEVERITY:
		return "Unset"
	case v1.Severity_LOW_SEVERITY:
		return "Low"
	case v1.Severity_MEDIUM_SEVERITY:
		return "Medium"
	case v1.Severity_HIGH_SEVERITY:
		return "High"
	case v1.Severity_CRITICAL_SEVERITY:
		return "Critical"
	default:
		panic("The severity enum has been updated, but this switch statement hasn't")
	}
}

// StringViolations converts []*v1.Policy_Violation to []string
func StringViolations(policyViolations []*v1.Policy_Violation) []string {
	violations := make([]string, 0, len(policyViolations))
	for _, p := range policyViolations {
		violations = append(violations, p.Message)
	}
	return violations
}
