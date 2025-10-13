package services

import (
	"fmt"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/stringutils"
)

// ServiceTypeToSlugName converts a service type (such as storage.ADMISSION_CONTROL_SERVICE) to the
// slug-case name of the service ('admission-control').
//
// Returns the empty string for invalid service types.
func ServiceTypeToSlugName(ty storage.ServiceType) string {
	tyName := ty.String()
	if !stringutils.ConsumeSuffix(&tyName, "_SERVICE") {
		return ""
	}
	tyName = strings.ToLower(tyName)
	tyName = strings.ReplaceAll(tyName, "_", "-")
	return tyName
}

// SlugNameToServiceType converts a "service slug name", e.g. "admission-control", to the
// corresponding ServiceType identifier, e.g. ServiceType_ADMISSION_CONTROL_SERVICE.
//
// This is done by
// - uppercasing
// - replacing dashes with underscores
// - appending the "_SERVICE" suffix.
//
// Returns ServiceType_UNKNOWN_SERVICE (0) in case the provided service slug-name
// representation does not correspond to a valid service type.
func SlugNameToServiceType(tyName string) storage.ServiceType {
	tyName = strings.ToUpper(tyName)
	tyName = strings.ReplaceAll(tyName, "-", "_")
	tyName = fmt.Sprintf("%s_SERVICE", tyName)
	return storage.ServiceType(storage.ServiceType_value[tyName])
}
