// Package cpecompat handles CPE values emitted by older StackRox components.
package cpecompat

import "github.com/quay/claircore/toolkit/types/cpe"

const (
	legacyUnsetCPE = "cpe:2.3:*"
	// UnsetCPE is the complete wildcard CPE used for unset values on the wire.
	UnsetCPE = "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
)

// Unbind expands the exact shortened wildcard CPE emitted by older Sensors to
// the complete wildcard before parsing it. All other inputs, including empty
// strings, are passed unchanged to cpe.Unbind, with its result and error returned.
func Unbind(s string) (cpe.WFN, error) {
	if s == legacyUnsetCPE {
		s = UnsetCPE
	}
	return cpe.Unbind(s)
}
