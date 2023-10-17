package observe

import (
	"fmt"

	"github.com/stackrox/rox/pkg/sac/effectiveaccessscope"
	"github.com/stackrox/rox/pkg/sync"
	"github.com/stackrox/rox/pkg/timestamp"
)

// ScopeCheckerCoreType is a string representation of a set of existing
// ScopeCheckerCore implementations.
type ScopeCheckerCoreType string

// This block declares string constants for ScopeCheckerCore types.
const (
	ScopeCheckerNone                 ScopeCheckerCoreType = "none"
	ScopeCheckerDenyForNoID          ScopeCheckerCoreType = "deny all (no identity)"
	ScopeCheckerAllowAdminAndService ScopeCheckerCoreType = "allow all (admin or service)"
	ScopeCheckerBuiltIn              ScopeCheckerCoreType = "built-in"
)

// RecordScopeCheckerCoreType writes the given scope checker type iff the
// receiver is not nil.
func (t *AuthzTrace) RecordScopeCheckerCoreType(sccType ScopeCheckerCoreType) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.sccType = sccType
}

// RecordKnownClustersAndNamespaces writes the number of all known clusters and
// namespaces iff the receiver is not nil.
func (t *AuthzTrace) RecordKnownClustersAndNamespaces(
	clusters []effectiveaccessscope.ClusterForSAC,
	namespaces []effectiveaccessscope.NamespaceForSAC,
) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.numClusters = int32(len(clusters))
	t.numNamespaces = int32(len(namespaces))

	// Compute clusterID -> clusterName for human-friendly traces.
	t.clusterNameByID = make(map[string]string, len(clusters))
	for _, cluster := range clusters {
		t.clusterNameByID[cluster.GetID()] = cluster.GetName()
	}
}

// RecordAllowOnResourceLevel writes an "allow resource" authorizer decision iff
// the receiver is not nil.
func (t *AuthzTrace) RecordAllowOnResourceLevel(access string, resourceName string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.allowed[fmt.Sprintf("%s for %s in *", access, resourceName)]++
}

// RecordDenyOnResourceLevel writes a "deny resource" authorizer decision iff
// the receiver is not nil.
func (t *AuthzTrace) RecordDenyOnResourceLevel(access string, resourceName string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.denied[fmt.Sprintf("%s for %s in *", access, resourceName)]++
}

// RecordAllowOnScopeLevel writes an "allow resource object in scope" authorizer
// decision iff the receiver is not nil.
func (t *AuthzTrace) RecordAllowOnScopeLevel(access string, resourceName string, clusterID string, namespaceName string, roleName string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Use cluster name to record a human-friendly trace.
	clusterName := t.clusterNameByID[clusterID]
	if clusterName == "" {
		clusterName = clusterID
	}

	t.allowed[fmt.Sprintf("%s for %s in '%s::%s' [%s]", access, resourceName, clusterName, namespaceName, roleName)]++
}

// RecordDenyOnScopeLevel writes a "deny resource object in scope" authorizer
// decision iff the receiver is not nil.
func (t *AuthzTrace) RecordDenyOnScopeLevel(access string, resourceName string, clusterID string, namespaceName string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Use cluster name to record a human-friendly trace.
	clusterName := t.clusterNameByID[clusterID]
	if clusterName == "" {
		clusterName = clusterID
	}

	t.denied[fmt.Sprintf("%s for %s in '%s::%s'", access, resourceName, clusterName, namespaceName)]++
}

// RecordEffectiveAccessScope writes the effective access scope for a scope
// represented by its name or ID.
func (t *AuthzTrace) RecordEffectiveAccessScope(accessScopeID string, accessScopeName string, eas string) {
	if t == nil {
		return
	}

	t.mutex.Lock()
	defer t.mutex.Unlock()

	// If for whatever reason access scope name is not set, fall back to ID.
	key := accessScopeName
	if key == "" {
		key = accessScopeID
	}
	t.effectiveAccessScopes[key] = eas
}

// AuthzTrace collects data related to authorization decisions made by a
// ScopeCheckerCore instance.
type AuthzTrace struct {
	arrivedAt timestamp.MicroTS

	sccType               ScopeCheckerCoreType
	numClusters           int32
	numNamespaces         int32
	denied                map[string]int32
	allowed               map[string]int32
	effectiveAccessScopes map[string]string

	// Auxiliary fields.
	mutex           sync.Mutex
	clusterNameByID map[string]string
}

// NewAuthzTrace creates an empty instance of AuthzTrace.
func NewAuthzTrace() *AuthzTrace {
	return &AuthzTrace{
		arrivedAt:             timestamp.Now(),
		denied:                make(map[string]int32),
		allowed:               make(map[string]int32),
		effectiveAccessScopes: make(map[string]string),
	}
}
