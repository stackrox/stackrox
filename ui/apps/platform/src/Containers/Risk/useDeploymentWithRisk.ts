import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import type { UseRestQueryReturn } from 'hooks/useRestQuery';
import { fetchDeploymentWithRisk } from 'services/DeploymentsService';
import type { DeploymentWithRisk } from 'services/DeploymentsService';

export default function useDeploymentWithRisk(
    deploymentId: string
): UseRestQueryReturn<DeploymentWithRisk> {
    const requestFn = useCallback(() => {
        return fetchDeploymentWithRisk(deploymentId);
    }, [deploymentId]);
    return useRestQuery(requestFn);
}
