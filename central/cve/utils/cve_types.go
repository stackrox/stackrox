package utils

import (
	"slices"

	"github.com/stackrox/rox/generated/storage"
)

// AddCVETypeIfAbsent adds the given CVE type to the given slice of CVE types if the slice does
// not already have the CVE type and returns a slice with the given type included.
func AddCVETypeIfAbsent(cveTypes []storage.CVE_CVEType, toAdd storage.CVE_CVEType) []storage.CVE_CVEType {
	// New CVE's types slice will be nil/empty.
	// Populate with the current CVE's.
	addToCVETypes := !slices.Contains(cveTypes, toAdd)
	// Add the new CVE's type to the type slice if it's not already in it.
	if addToCVETypes {
		return append(cveTypes, toAdd)
	}

	return cveTypes
}
