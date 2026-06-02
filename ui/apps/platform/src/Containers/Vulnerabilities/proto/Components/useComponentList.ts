import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoComponentListItem = {
    name: string;
    versionCount: number;
    cveCount: number;
    imageCount: number;
    topSeverity: number;
    topCvss: number;
    criticalCount: number;
    importantCount: number;
    moderateCount: number;
    lowCount: number;
};

type ComponentListResponse = {
    components: ProtoComponentListItem[];
    totalCount: number;
};

/**
 * Fetches the prototype component list from the REST API.
 */
export function useComponentList(
    limit = 50,
    offset = 0,
    sortBy = 'severity',
    sortDir = 'desc'
) {
    const [data, setData] = useState<ComponentListResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<ComponentListResponse>(
                `/v1/scandata/components?limit=${limit}&offset=${offset}&sortBy=${sortBy}&sortDir=${sortDir}`
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
