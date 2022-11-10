import { useEffect, useState } from 'react';
import { fetchDeployment } from 'services/DeploymentsService';

import { Deployment } from 'types/deployment.proto';

type Result = { isLoading: boolean; deployment: Deployment | null; error: string | null };

const defaultResultState = { deployment: null, error: null, isLoading: true };

/*
 * This hook does an API call to the deployment API to get a deployment
 */
function useFetchDeployment(deploymentId: string): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        if (deploymentId) {
            fetchDeployment(deploymentId)
                .then((data) => {
                    setResult({ deployment: data || null, error: null, isLoading: false });
                })
                .catch((error) => {
                    setResult({ deployment: null, error, isLoading: false });
                });
        }
    }, [deploymentId]);

    return result;
}

export default useFetchDeployment;
