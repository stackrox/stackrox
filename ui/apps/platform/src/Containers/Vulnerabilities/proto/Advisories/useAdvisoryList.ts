import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoAdvisoryListItem = {
    advisoryId: string;
    cveName: string;
    severity: number;
    cvss: number;
    sourceName: string;
    description: string;
    fixedBy?: string;
    imageCount: number;
    link: string;
};

type AdvisoryListResponse = {
    advisories: ProtoAdvisoryListItem[];
    totalCount: number;
};

/**
 * Fetches the prototype advisory list from the REST API.
 */
export function useAdvisoryList(limit = 50, offset = 0) {
    const [data, setData] = useState<AdvisoryListResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<AdvisoryListResponse>(
                `/v1/scandata/advisories?limit=${limit}&offset=${offset}`
            )
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [limit, offset]);

    return { data, loading, error };
}
