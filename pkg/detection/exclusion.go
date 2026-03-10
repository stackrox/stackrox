package detection

import (
	"context"
	"strings"

	"github.com/stackrox/rox/generated/storage"
	"github.com/stackrox/rox/pkg/protocompat"
	"github.com/stackrox/rox/pkg/scopecomp"
)

func auditEventMatchesExclusions(ctx context.Context, auditEvent *storage.KubernetesEvent, exclusions []*compiledExclusion) bool {
	for _, exclusion := range exclusions {
		if exclusion.MatchesAuditEvent(ctx, auditEvent) {
			return true
		}
	}
	return false
}

func auditEventMatchesScopes(ctx context.Context, auditEvent *storage.KubernetesEvent, scopes []*scopecomp.CompiledScope) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		if scope.MatchesAuditEvent(ctx, auditEvent) {
			return true
		}
	}
	return false
}

func deploymentMatchesExclusions(ctx context.Context, deployment *storage.Deployment, exclusions []*compiledExclusion) bool {
	for _, exclusion := range exclusions {
		if exclusion.MatchesDeployment(ctx, deployment) {
			return true
		}
	}
	return false
}

func deploymentMatchesScopes(ctx context.Context, deployment *storage.Deployment, scopes []*scopecomp.CompiledScope) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		if scope.MatchesDeployment(ctx, deployment) {
			return true
		}
	}
	return false
}

func matchesImageExclusion(image string, policy *storage.Policy) bool {
	for _, w := range policy.GetExclusions() {
		if w.GetImage() == nil {
			continue
		}
		if exclusionIsExpired(w) {
			continue
		}
		// The rationale for using a prefix is that it is the easiest way in the current format
		// to support excluding registries, registry/remote, etc
		if strings.HasPrefix(image, w.GetImage().GetName()) {
			return true
		}
	}
	return false
}

func exclusionIsExpired(exclusion *storage.Exclusion) bool {
	// If they don't set an expiration time, the excluded scope never expires.
	if exclusion.GetExpiration() == nil {
		return false
	}
	return protocompat.CompareTimestamps(exclusion.GetExpiration(), protocompat.TimestampNow()) < 0
}
