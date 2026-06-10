import { useEffect, useState } from 'react';

import axios from 'services/instance';

export type ComponentCVE = {
    cveName: string;
    severity: number;
    cvss: number;
    fixable: boolean;
    fixedBy?: string;
    description?: string;
    imageCount: number;
};

type ComponentCVEsResponse = {
    componentName: string;
    componentVersion: string;
    cves: ComponentCVE[];
};

/**
 * Fetches CVEs for a specific component name+version from the REST API.
 * Only fetches when `enabled` is true (i.e. when the row is expanded).
 */
export function useComponentCVEs(componentName: string, componentVersion: string, enabled: boolean) {
    const [data, setData] = useState<ComponentCVE[]>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        if (!enabled || !componentName || !componentVersion) {
            return;
        }
        setLoading(true);
        axios
            .get<ComponentCVEsResponse>(
                `/v1/scandata/components/cves?name=${encodeURIComponent(componentName)}&version=${encodeURIComponent(componentVersion)}`
            )
            .then((res) => {
                setData(res.data.cves ?? []);
                setError(null);
            })
            .catch((err: Error) => setError(err))
            .finally(() => setLoading(false));
    }, [componentName, componentVersion, enabled]);

    return { data, loading, error };
}
