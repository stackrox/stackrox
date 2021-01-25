package centralsensor

import (
	"sort"
)

// SensorCapability identifies a capability exposed by sensor.
type SensorCapability string

//go:generate genny -in=../set/generic.go -out=gen-caps-set.go -pkg centralsensor gen "KeyType=SensorCapability"

// String returns the string form of sensor capability.
func (s SensorCapability) String() string {
	return string(s)
}

// CapSetFromStringSlice takes a slice of strings, and converts it into a SensorCapabilitySet.
func CapSetFromStringSlice(capStrs ...string) SensorCapabilitySet {
	capSet := NewSensorCapabilitySet()
	for _, capStr := range capStrs {
		capSet.Add(SensorCapability(capStr))
	}
	return capSet
}

// CapSetToStringSlice takes a capability set, and converts it into a string slice.
func CapSetToStringSlice(capSet SensorCapabilitySet) []string {
	strs := make([]string, 0, len(capSet))
	for capability := range capSet {
		strs = append(strs, capability.String())
	}
	sort.Strings(strs)
	return strs
}
