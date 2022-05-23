import { useCallback } from 'react';
import useURLParameter from 'hooks/useURLParameter';

// TODO We will likely want to make this support multiple cluster ids in the future.
export default function useURLCluster(defaultClusterId: string) {
    const [cluster, setClusterInternal] = useURLParameter('cluster', defaultClusterId || undefined);
    const setCluster = useCallback(
        (clusterId?: string) => {
            setClusterInternal(clusterId);
        },
        [setClusterInternal]
    );

    return {
        cluster: typeof cluster === 'string' ? cluster : undefined,
        setCluster,
    };
}
