import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoAdvisory = {
    id: string;
    severity: number;
    cvss: number;
    sourceName: string;
    description?: string;
    link?: string;
    fixedBy?: string;
};

export type ProtoComponent = {
    name: string;
    version: string;
    source: string;
    fixedBy: string;
    imageCount: number;
};

export type ProtoImageComponent = {
    name: string;
    version: string;
    source: string;
    fixedBy?: string;
    advisories?: string[];
};

export type ProtoImage = {
    imageId: string;
    imageUuid?: string;
    imageName?: string;
    componentCount: number;
    severity: number;
    fixable: boolean;
    components: ProtoImageComponent[];
};

export type CveDetailResponse = {
    cveName: string;
    severity: number;
    cvss: number;
    advisories: ProtoAdvisory[];
    components: ProtoComponent[];
    images: ProtoImage[];
};

/**
 * Fetches prototype CVE detail from the REST API.
 */
export function useCveDetail(cveName: string) {
    const [data, setData] = useState<CveDetailResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        if (!cveName) {
            return;
        }
        setLoading(true);
        axios
            .get<CveDetailResponse>(
                `/v1/scandata/cves/${encodeURIComponent(cveName)}`
            )
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [cveName]);

    return { data, loading, error };
}
