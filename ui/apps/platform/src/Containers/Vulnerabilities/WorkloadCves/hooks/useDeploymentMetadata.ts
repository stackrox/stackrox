import { useCallback } from 'react';
import { fetchDeployment } from 'services/DeploymentsService';
import { fetchImagesCount } from 'services/imageService';
import useRestQuery from 'hooks/useRestQuery';
import { wrapInQuotes } from 'utils/searchUtils';
import { isNotFoundError } from 'utils/responseErrorUtils';

import type { DeploymentMetadata } from '../Deployment/DeploymentPageHeader';

async function fetchDeploymentMetadata(id: string): Promise<DeploymentMetadata | null> {
    try {
        const [deployment, imageCount] = await Promise.all([
            fetchDeployment(id),
            fetchImagesCount(`Deployment ID:${wrapInQuotes(id)}`),
        ]);

        return {
            id: deployment.id,
            name: deployment.name,
            namespace: deployment.namespace,
            clusterName: deployment.clusterName,
            created: deployment.created ?? null,
            imageCount,
        };
    } catch (error) {
        if (isNotFoundError(error)) {
            return null;
        }
        throw error;
    }
}

export default function useDeploymentMetadata(deploymentId: string) {
    const requestFn = useCallback(() => fetchDeploymentMetadata(deploymentId), [deploymentId]);
    const { data, isLoading, error } = useRestQuery(requestFn);

    const isCurrent = data === undefined || data === null || data.id === deploymentId;
    const deployment = isCurrent ? data : undefined;

    return {
        data: deployment === undefined ? undefined : { deployment },
        loading: isLoading && deployment === undefined,
        error: isNotFoundError(error) ? undefined : error,
    };
}
