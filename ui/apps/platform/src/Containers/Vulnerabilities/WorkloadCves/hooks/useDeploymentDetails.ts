import { useCallback } from 'react';
import { fetchDeployment } from 'services/DeploymentsService';
import useRestQuery from 'hooks/useRestQuery';
import { isNotFoundError } from 'utils/responseErrorUtils';

import type { Deployment } from 'types/deployment.proto';

export type DeploymentDetails = {
    id: string;
    name: string;
    cluster: {
        id: string;
        name: string;
    } | null;
    namespace: string;
    replicas: number;
    created: string | null;
    serviceAccount: string;
    type: string;
    labels: {
        key: string;
        value: string;
    }[];
    annotations: {
        key: string;
        value: string;
    }[];
};

function mapRecordToKeyValues(
    record: Record<string, string> | undefined
): { key: string; value: string }[] {
    return Object.entries(record ?? {}).map(([key, value]) => ({ key, value }));
}

export function mapDeploymentToDetails(deployment: Deployment): DeploymentDetails {
    return {
        id: deployment.id,
        name: deployment.name,
        cluster: deployment.clusterId
            ? { id: deployment.clusterId, name: deployment.clusterName }
            : null,
        namespace: deployment.namespace,
        replicas: Number(deployment.replicas) || 0,
        created: deployment.created ?? null,
        serviceAccount: deployment.serviceAccount,
        type: deployment.type,
        labels: mapRecordToKeyValues(deployment.labels),
        annotations: mapRecordToKeyValues(deployment.annotations),
    };
}

async function fetchDeploymentDetails(id: string): Promise<DeploymentDetails | null> {
    try {
        return mapDeploymentToDetails(await fetchDeployment(id));
    } catch (error) {
        if (isNotFoundError(error)) {
            return null;
        }
        throw error;
    }
}

export default function useDeploymentDetails(deploymentId: string) {
    const requestFn = useCallback(() => fetchDeploymentDetails(deploymentId), [deploymentId]);
    const { data, isLoading, error } = useRestQuery(requestFn);

    const isCurrent = data === undefined || data === null || data.id === deploymentId;
    const deployment = isCurrent ? data : undefined;

    return {
        data: deployment === undefined ? undefined : { deployment },
        previousData: undefined as { deployment: DeploymentDetails } | undefined,
        loading: isLoading && deployment === undefined,
        error: isNotFoundError(error) ? undefined : error,
    };
}
