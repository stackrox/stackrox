import { fetchDeploymentRiskSummary } from 'services/DeploymentsService';
import type { DeploymentRiskSummary } from 'services/DeploymentsService';

// Session-scoped cache for AI risk summaries, keyed by deployment id. Generating a
// summary is expensive and the result should be relatively stable for a deployment within a browser
// session, so resolved summaries are cached and in-flight requests are deduped at
// module scope. The cache therefore persists across page navigations (until a full
// reload) - the behavior a query library would provide,
// without the dependency. Failures are not cached, so a later attempt retries.

const resolved = new Map<string, DeploymentRiskSummary>();
const inFlight = new Map<string, Promise<DeploymentRiskSummary>>();

/** Fetches the summary, returning the cached value or an in-flight request when available. */
export function fetchCachedDeploymentRiskSummary(
    deploymentId: string
): Promise<DeploymentRiskSummary> {
    const cached = resolved.get(deploymentId);
    if (cached) {
        return Promise.resolve(cached);
    }
    const pending = inFlight.get(deploymentId);
    if (pending) {
        return pending;
    }
    const request = fetchDeploymentRiskSummary(deploymentId)
        .then((summary) => {
            resolved.set(deploymentId, summary);
            return summary;
        })
        .finally(() => {
            inFlight.delete(deploymentId);
        });
    inFlight.set(deploymentId, request);
    return request;
}

/**
 * Synchronous read of an already-cached summary, if any. Lets a freshly mounted
 * component render a previously fetched summary immediately without re-investigating.
 */
export function peekCachedDeploymentRiskSummary(
    deploymentId: string
): DeploymentRiskSummary | undefined {
    return resolved.get(deploymentId);
}
