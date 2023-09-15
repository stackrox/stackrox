package centralsensor

import (
	"sort"

	"github.com/stackrox/rox/pkg/set"
)

// SensorCapability identifies a capability exposed by sensor.
type SensorCapability string

// String returns the string form of sensor capability.
func (s SensorCapability) String() string {
	return string(s)
}

// CapSetFromStringSlice takes a slice of strings, and converts it into a SensorCapabilitySet.
func CapSetFromStringSlice(capStrs ...string) set.Set[SensorCapability] {
	capSet := set.NewSet[SensorCapability]()
	for _, capStr := range capStrs {
		capSet.Add(SensorCapability(capStr))
	}
	return capSet
}

// CapSetToStringSlice takes a capability set, and converts it into a string slice.
func CapSetToStringSlice(capSet set.Set[SensorCapability]) []string {
	strs := make([]string, 0, len(capSet))
	for capability := range capSet {
		strs = append(strs, capability.String())
	}
	sort.Strings(strs)
	return strs
}

// TODO: Generics

// CentralCapability identifies a capability exposed by central.
type CentralCapability string

// String returns the string form of central capability.
func (c CentralCapability) String() string {
	return string(c)
}

// CentralCapSetFromStringSlice takes a slice of strings, and converts it into a SensorCapabilitySet.
func CentralCapSetFromStringSlice(capStrs ...string) set.Set[CentralCapability] {
	capSet := set.NewSet[CentralCapability]()
	for _, capStr := range capStrs {
		capSet.Add(CentralCapability(capStr))
	}
	return capSet
}

// CentralCapSetToStringSlice takes a capability set, and converts it into a string slice.
func CentralCapSetToStringSlice(capSet set.Set[CentralCapability]) []string {
	strs := make([]string, 0, len(capSet))
	for capability := range capSet {
		strs = append(strs, capability.String())
	}
	sort.Strings(strs)
	return strs
}
