import { useEffect, useState } from 'react';

import { AccessScope, fetchAccessScopes } from 'services/AccessScopesService';

type Result = { isLoading: boolean; scopes: AccessScope[]; error: string | null };

const defaultResultState = { scopes: [], error: null, isLoading: true };

/*
 * This hook does an API call to the baseline status API to get the baseline status
 * of the supplied peers
 */
function useFetchScopes(): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        fetchAccessScopes()
            .then((data) => {
                setResult({ scopes: data || null, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ scopes: [], error, isLoading: false });
            });
    }, []);

    return result;
}

export default useFetchScopes;
