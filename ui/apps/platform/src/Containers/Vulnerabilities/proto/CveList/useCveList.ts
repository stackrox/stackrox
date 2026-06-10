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
    sortDir = 'desc',
    cveFilter = ''
) {
    const [data, setData] = useState<CveListResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        setLoading(true);
        const params = new URLSearchParams({
            limit: String(limit),
            offset: String(offset),
            sortBy,
            sortDir,
        });
        if (cveFilter) {
            params.set('cve', cveFilter);
        }
        axios
            .get<CveListResponse>(`/v1/scandata/cves?${params.toString()}`)
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [limit, offset, sortBy, sortDir, cveFilter]);

    return { data, loading, error };
}
