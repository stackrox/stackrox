package services

import (
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// ServiceTypeToSlugName converts a service type (such as storage.ADMISSION_CONTROL_SERVICE) to the
// slug-case name of the service ('admission-control').
func ServiceTypeToSlugName(ty storage.ServiceType) string {
	tyName := ty.String()
	if !stringutils.ConsumeSuffix(&tyName, "_SERVICE") {
		return ""
	}
	tyName = strings.ToLower(tyName)
	tyName = strings.Replace(tyName, "_", "-", -1)
	return tyName
}
