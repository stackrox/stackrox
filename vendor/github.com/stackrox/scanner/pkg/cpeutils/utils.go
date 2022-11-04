package cpeutils

import (
	"strings"

	"github.com/facebookincubator/nvdtools/wfn"
)

// GetMostSpecificCPE deterministically returns the CPE that is the most specific
// from the set of matches. This function requires that len(cpes) > 0
func GetMostSpecificCPE(cpes []wfn.AttributesWithFixedIn) wfn.AttributesWithFixedIn {
	mostSpecificCPE := cpes[0]
	for _, cpe := range cpes[1:] {
		if compareAttributes(cpe, mostSpecificCPE) > 0 {
			mostSpecificCPE = cpe
		}
	}
	return mostSpecificCPE
}

func compareAttributes(c1, c2 wfn.AttributesWithFixedIn) int {
	if cmp := strings.Compare(c1.Vendor, c2.Vendor); cmp != 0 {
		return cmp
	}
	if cmp := strings.Compare(c1.Product, c2.Product); cmp != 0 {
		return cmp
	}
	return strings.Compare(c1.Version, c2.Version)
}

// IsOpenShiftCPE returns if the passed CPE is a known OpenShift CPE
func IsOpenShiftCPE(cpe string) bool {
	return strings.HasPrefix(cpe, "cpe:/a:redhat:openshift:")
}
