package standards

import (
	"github.com/stackrox/rox/pkg/errox"
)

// GetSupportedStandards returns the ids of all supported compliance standards
func GetSupportedStandards() []string {
	var registeredStandards []string
	for _, standard := range RegistrySingleton().AllStandards() {
		registeredStandards = append(registeredStandards, standard.ID)
	}
	return registeredStandards
}

// FilterSupported filters given standards into supported standards and unsupported standards
func FilterSupported(standardIDs []string) (supported []string, unsupported []string) {
	for _, standardID := range standardIDs {
		if IsSupported(standardID) {
			supported = append(supported, standardID)
		} else {
			unsupported = append(unsupported, standardID)
		}
	}
	return
}

// IsSupported returns true if the compliance standard is supported
func IsSupported(standardID string) bool {
	return RegistrySingleton().LookupStandard(standardID) != nil
}

// UnSupportedStandardsErr builds error message for unsupported compliance standards and returns the error
func UnSupportedStandardsErr(unsupported ...string) error {
	return errox.InvalidArgs.CausedByf("unsupported standard(s): %+v. Supported standards are %+v", unsupported, GetSupportedStandards())
}
