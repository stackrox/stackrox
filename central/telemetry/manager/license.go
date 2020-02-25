package manager

import (
	"regexp"

	licenseproto "github.com/stackrox/rox/generated/shared/license"
)

var (
	stackroxLicenseeIDRegex = regexp.MustCompile(`^[^@]+@(stackrox\.com|stackrox-[^.]+\.iam\.gserviceaccount\.com)$`)
)

func isStackRoxLicense(licenseMD *licenseproto.License_Metadata) bool {
	return stackroxLicenseeIDRegex.MatchString(licenseMD.GetLicensedForId())
}
