import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ProtoComponentImage = {
    imageId: string;
    imageUuid?: string;
    imageName?: string;
    version: string;
    arch?: string;
    cveCount: number;
    topSeverity: number;
    fixable: boolean;
};

type ComponentImagesResponse = {
    images: ProtoComponentImage[];
};

/**
 * Fetches images containing the given component from the REST API.
 */
export function useComponentImages(componentName: string) {
    const [data, setData] = useState<ProtoComponentImage[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        if (!componentName) {
            return;
        }
        setLoading(true);
        axios
            .get<ComponentImagesResponse>(
                `/v1/scandata/components/${encodeURIComponent(componentName)}/images`
            )
            .then((res) => {
                setData(res.data.images ?? []);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [componentName]);

    return { data, loading, error };
}
