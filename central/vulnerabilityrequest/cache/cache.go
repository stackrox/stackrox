package cache

import (
	"github.com/stackrox/rox/generated/storage"
)

// VulnReqCache provides functionality to cache vulnerability requests.
//
//go:generate mockgen-wrapper
type VulnReqCache interface {
	Add(request *storage.VulnerabilityRequest) bool
	AddMany(requests ...*storage.VulnerabilityRequest)
	Remove(requestID string) bool
	RemoveMany(requestIDs ...string) bool
	// GetVulnsWithState returns that effective target state for all cves in the given scope.
	GetVulnsWithState(registry, remote, tag string) map[string]storage.VulnerabilityState
	// GetEffectiveVulnStateForImage returns the effective state of the vulnerabilities within the given image.
	GetEffectiveVulnStateForImage(cves []string, registry, remote, tag string) map[string]storage.VulnerabilityState
	// GetEffectiveVulnReqIDForImage returns the vuln request in effect on given image+cve combination.
	GetEffectiveVulnReqIDForImage(registry, remote, tag, cve string) string
}
