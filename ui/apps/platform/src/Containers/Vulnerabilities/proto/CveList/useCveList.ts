import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoCVEListItem = {
    cveName: string;
    severity: number;
    cvss: number;
    imageCount: number;
    fixable: boolean;
    firstSeen: string | null;
    publishedDate?: string;
    epssProbability?: number;
};

type CveListResponse = {
    cves: ProtoCVEListItem[];
    totalCount: number;
};

/**
 * Fetches the prototype CVE list from the REST API.
 */
export function useCveList(
    limit = 50,
    offset = 0,
    sortBy = 'severity',
    sortDir = 'desc'
) {
    const [data, setData] = useState<CveListResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<CveListResponse>(
                `/v1/scandata/cves?limit=${limit}&offset=${offset}&sortBy=${sortBy}&sortDir=${sortDir}`
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
