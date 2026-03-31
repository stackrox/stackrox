import useURLStringUnion from 'hooks/useURLStringUnion';
import { deploymentStatuses } from 'types/deploymentStatus';
import type { DeploymentStatus } from 'types/deploymentStatus';

/**
 * Reads the `deploymentStatus` URL parameter, defaulting to 'DEPLOYED'.
 * Read-only — the URL parameter is mutated by the filter component, not by this hook.
 */
export default function useDeploymentStatus(): DeploymentStatus {
    const [status] = useURLStringUnion('deploymentStatus', deploymentStatuses);
    return status;
}
