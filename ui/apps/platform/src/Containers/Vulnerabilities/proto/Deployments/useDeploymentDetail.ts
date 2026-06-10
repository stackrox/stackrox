import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoDeploymentImage = {
    imageId: string;
    imageName: string;
    cveCount: number;
    topSeverity: number;
    fixable: boolean;
};

export type DeploymentDetailResponse = {
    id: string;
    name: string;
    cluster: string;
    namespace: string;
    images: ProtoDeploymentImage[];
};

/**
 * Fetches prototype deployment detail from the REST API.
 */
export function useDeploymentDetail(deploymentId: string) {
    const [data, setData] = useState<DeploymentDetailResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        if (!deploymentId) {
            return;
        }
        setLoading(true);
        axios
            .get<DeploymentDetailResponse>(
                `/v1/scandata/deployments/${encodeURIComponent(deploymentId)}`
            )
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [deploymentId]);

    return { data, loading, error };
}
