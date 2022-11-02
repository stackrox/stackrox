package common

import (
	"log"
)

// (sigh, only if using reflection could be deemed "idigomatic"...)
// Also don't put these into pkg/common/utils/util.go since utils package should not
// depend on application specifics (for example, this common package).

// RgnSvcPairSliceRemove removes an element from a RegionServicePair slice at the specified index
func RgnSvcPairSliceRemove(in []*RegionServicePair, i int) []*RegionServicePair {
	if i < 0 || i >= len(in) {
		log.Panicf("Index out of bound: %d", i)
	}
	in[i] = in[len(in)-1]
	return in[:len(in)-1]
}

// SvcIPRangesSliceRemove removes an element from a ServiceIPRanges slice at the specified index
func SvcIPRangesSliceRemove(in []*ServiceIPRanges, i int) []*ServiceIPRanges {
	if i < 0 || i >= len(in) {
		log.Panicf("Index out of bound: %d", i)
	}
	in[i] = in[len(in)-1]
	return in[:len(in)-1]
}

// RgnNetDetSliceRemove removes an element from a RegionNetworkDetail slice at the specified index
func RgnNetDetSliceRemove(in []*RegionNetworkDetail, i int) []*RegionNetworkDetail {
	if i < 0 || i >= len(in) {
		log.Panicf("Index out of bound: %d", i)
	}
	in[i] = in[len(in)-1]
	return in[:len(in)-1]
}
