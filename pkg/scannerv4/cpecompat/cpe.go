// Package cpecompat handles CPE values emitted by older StackRox components.
package cpecompat

import "github.com/quay/claircore/toolkit/types/cpe"

const (
	legacyUnsetCPE = "cpe:2.3:*"
	// UnsetCPE is the complete wildcard CPE used for unset values on the wire.
	UnsetCPE = "cpe:2.3:*:*:*:*:*:*:*:*:*:*:*"
)

// Unbind accepts the shortened wildcard CPE emitted by older Sensors for the
// synthetic RHCOS package. ClairCore toolkit v1.7.0 rejects that string because
// it does not contain all eleven CPE attributes. Its missing attributes were
// unset in the older parser, which CPE matching treats like wildcards.
func Unbind(s string) (cpe.WFN, error) {
	if s == legacyUnsetCPE {
		s = UnsetCPE
	}
	return cpe.Unbind(s)
}
