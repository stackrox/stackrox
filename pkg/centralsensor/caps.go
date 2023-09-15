package centralsensor

import (
	"sort"

	"github.com/stackrox/rox/pkg/set"
)

// CentralCapability identifies a capability exposed by central.
type CentralCapability string

// SensorCapability identifies a capability exposed by sensor.
type SensorCapability string

// CapSetFromStringSlice takes a slice of strings, and converts it into a CapabilitySet.
func CapSetFromStringSlice[V CentralCapability | SensorCapability](capStrs ...string) set.Set[V] {
	capSet := set.NewSet[V]()
	for _, capStr := range capStrs {
		capSet.Add(V(capStr))
	}
	return capSet
}

// CapSetToStringSlice takes a CapabilitySet, and converts it into a string slice.
func CapSetToStringSlice[V CentralCapability | SensorCapability](capSet set.Set[V]) []string {
	strs := make([]string, 0, len(capSet))
	for capability := range capSet {
		strs = append(strs, string(capability))
	}
	sort.Strings(strs)
	return strs
}
