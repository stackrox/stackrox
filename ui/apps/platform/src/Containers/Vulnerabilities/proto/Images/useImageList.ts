import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoImageListItem = {
    imageId: string;
    imageUuid: string;
    imageName: string;
    imageOS: string;
    cveCount: number;
    componentCount: number;
    topSeverity: number;
    topCvss: number;
    fixable: boolean;
    scanTime: string | null;
    criticalCount: number;
    importantCount: number;
    moderateCount: number;
    lowCount: number;
};

type ImageListResponse = {
    images: ProtoImageListItem[];
    totalCount: number;
};

/**
 * Fetches the prototype image list from the REST API.
 */
export function useImageList(
    limit = 50,
    offset = 0,
    sortBy = 'severity',
    sortDir = 'desc'
) {
    const [data, setData] = useState<ImageListResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        setLoading(true);
        axios
            .get<ImageListResponse>(
                `/v1/scandata/images?limit=${limit}&offset=${offset}&sortBy=${sortBy}&sortDir=${sortDir}`
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
