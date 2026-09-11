import { useCallback } from 'react';

import { fetchAlertCount } from 'services/AlertsService';
import { fetchClusters } from 'services/ClustersService';
import { fetchDeploymentsCount } from 'services/DeploymentsService';
import { fetchImageCount } from 'services/imageService';
import { fetchSecretCount } from 'services/SecretsService';
import { fetchNodeCount } from 'services/SearchService';
import useRestQuery from 'hooks/useRestQuery';

export type TileResource = 'Cluster' | 'Node' | 'Alert' | 'Deployment' | 'Image' | 'Secret';

export type SummaryCountsResponse = {
    clusterCount?: number;
    nodeCount?: number;
    violationCount?: number;
    deploymentCount?: number;
    imageCount?: number;
    secretCount?: number;
};

async function fetchSummaryCounts(
    hasReadAccessForResource: Record<TileResource, boolean>
): Promise<SummaryCountsResponse> {
    const requests: Promise<Partial<SummaryCountsResponse>>[] = [];

    if (hasReadAccessForResource.Cluster) {
        requests.push(fetchClusters().then((clusters) => ({ clusterCount: clusters.length })));
    }
    if (hasReadAccessForResource.Node) {
        requests.push(fetchNodeCount().then((nodeCount) => ({ nodeCount })));
    }
    if (hasReadAccessForResource.Alert) {
        requests.push(fetchAlertCount({}).request.then((violationCount) => ({ violationCount })));
    }
    if (hasReadAccessForResource.Deployment) {
        requests.push(fetchDeploymentsCount({}).then((deploymentCount) => ({ deploymentCount })));
    }
    if (hasReadAccessForResource.Image) {
        requests.push(fetchImageCount().then((imageCount) => ({ imageCount })));
    }
    if (hasReadAccessForResource.Secret) {
        requests.push(fetchSecretCount({}).request.then((secretCount) => ({ secretCount })));
    }

    const parts = await Promise.all(requests);
    return Object.assign(
        {
            clusterCount: undefined,
            nodeCount: undefined,
            violationCount: undefined,
            deploymentCount: undefined,
            imageCount: undefined,
            secretCount: undefined,
        },
        ...parts
    ) as SummaryCountsResponse;
}

export default function useSummaryCounts(hasReadAccessForResource: Record<TileResource, boolean>) {
    const accessKey = [
        hasReadAccessForResource.Cluster,
        hasReadAccessForResource.Node,
        hasReadAccessForResource.Alert,
        hasReadAccessForResource.Deployment,
        hasReadAccessForResource.Image,
        hasReadAccessForResource.Secret,
    ].join(',');

    const restQuery = useCallback(
        () => fetchSummaryCounts(hasReadAccessForResource),
        // hasReadAccessForResource is represented by accessKey.
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [accessKey]
    );

    return useRestQuery(restQuery);
}
