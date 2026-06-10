import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoDeploymentListItem = {
    id: string;
    name: string;
    cluster: string;
    namespace: string;
    imageCount: number;
    cveCount: number;
    topSeverity: number;
    fixable: boolean;
};

type DeploymentListResponse = {
    deployments: ProtoDeploymentListItem[];
    totalCount: number;
};

/**
 * Fetches the prototype deployment list from the REST API.
 */
export function useDeploymentList(
    limit = 50,
    offset = 0,
    sortBy = 'severity',
    sortDir = 'desc'
) {
    const [data, setData] = useState<DeploymentListResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<DeploymentListResponse>(
                `/v1/scandata/deployments?limit=${limit}&offset=${offset}&sortBy=${sortBy}&sortDir=${sortDir}`
            )
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [limit, offset, sortBy, sortDir]);

    return { data, loading, error };
}
