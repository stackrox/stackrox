import { useEffect, useState } from 'react';
import { fetchDeployments } from 'services/DeploymentsService';
import { ListDeployment } from 'types/deployment.proto';
import { SearchFilter } from 'types/search';

export type UseDeploymentsAtRiskReturn = {
    deployments: ListDeployment[];
    loading: boolean;
    error: Error | null;
};

export default function useDeploymentsAtRisk(
    searchFilter: SearchFilter,
    numberOfResults = 6
): UseDeploymentsAtRiskReturn {
    const [deployments, setDeployments] = useState<ListDeployment[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        const { request, cancel } = fetchDeployments(
            searchFilter,
            { field: 'Deployment Risk Priority', reversed: 'false' },
            0,
            numberOfResults
        );

        setError(null);

        request
            .then((results) => {
                setDeployments(results.map(({ deployment }) => deployment));
                setLoading(false);
                setError(null);
            })
            .catch((err) => {
                setLoading(true);
                setError(err);
            });

        return cancel;
    }, [searchFilter, numberOfResults]);

    return { deployments, loading, error };
}
