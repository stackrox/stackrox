import { useCallback } from 'react';

import { fetchClusters } from 'services/ClustersService';
import { fetchNamespaces } from 'services/NamespacesService';
import useRestQuery from 'hooks/useRestQuery';

import type { Cluster } from '../types';

async function fetchClustersWithNamespaces(): Promise<Cluster[]> {
    const [clusters, namespaces] = await Promise.all([fetchClusters(), fetchNamespaces()]);
    return clusters.map((cluster) => ({
        id: cluster.id,
        name: cluster.name,
        namespaces: namespaces
            .filter((namespace) => namespace.metadata.clusterId === cluster.id)
            .map((namespace) => ({
                metadata: {
                    id: namespace.metadata.id,
                    name: namespace.metadata.name,
                },
            })),
    }));
}

export default function useClustersWithNamespaces() {
    const restQuery = useCallback(() => fetchClustersWithNamespaces(), []);
    return useRestQuery(restQuery);
}
