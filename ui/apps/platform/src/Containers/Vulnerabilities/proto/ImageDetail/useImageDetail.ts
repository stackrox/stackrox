import { useEffect, useState } from 'react';
import axios from 'services/instance';

export type ImageCVE = {
    cveName: string;
    severity: number;
    cvss: number;
    fixedBy?: string;
    advisories?: string[];
};

export type ImageComponent = {
    name: string;
    version: string;
    source: string;
    location?: string;
    cves: ImageCVE[];
};

export type CVESummary = {
    total: number;
    critical: number;
    important: number;
    moderate: number;
    low: number;
};

export type ImageDetailResponse = {
    imageId: string;
    imageName?: string;
    scanTime?: string;
    scannerVersion?: string;
    bundleVersion?: string;
    dataSources?: string[];
    components: ImageComponent[];
    cveSummary: CVESummary;
};

/**
 * Fetches image detail data from the REST scan-data endpoint.
 *
 * @param imageId - SHA digest of the image (e.g. "sha256:abc...")
 */
export function useImageDetail(imageId: string) {
    const [data, setData] = useState<ImageDetailResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        if (!imageId) {
            setLoading(false);
            return;
        }
        setLoading(true);
        axios
            .get<ImageDetailResponse>(
                `/v1/scandata/images/${encodeURIComponent(imageId)}`
            )
            .then((res) => {
                setData(res.data);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [imageId]);

    return { data, loading, error };
}
