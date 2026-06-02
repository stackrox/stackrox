import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoComponentVersion = {
    version: string;
    source: string;
    cveCount: number;
    imageCount: number;
    topSeverity: number;
    topCvss: number;
    fixable: boolean;
    fixedBy: string;
};

export type ComponentDetailResponse = {
    name: string;
    versions: ProtoComponentVersion[];
};

/**
 * Fetches prototype component detail from the REST API.
 */
export function useComponentDetail(componentName: string) {
    const [data, setData] = useState<ComponentDetailResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        if (!componentName) {
            return;
        }
        setLoading(true);
        axios
            .get<ComponentDetailResponse>(
                `/v1/scandata/components/detail?name=${encodeURIComponent(componentName)}`
            )
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [componentName]);

    return { data, loading, error };
}
