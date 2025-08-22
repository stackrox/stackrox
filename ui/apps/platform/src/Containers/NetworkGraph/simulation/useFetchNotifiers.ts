import { useEffect, useState } from 'react';

import {
    NotifierIntegrationBase,
    fetchNotifierIntegrations,
} from 'services/NotifierIntegrationsService';

type Result = { isLoading: boolean; notifiers: NotifierIntegrationBase[]; error: string | null };

const defaultResultState = { notifiers: [], error: null, isLoading: true };

/*
 * This hook does an API call to the notifiers API to get the list of notifiers
 */
function useFetchNotifiers(): Result {
    const [result, setResult] = useState<Result>(defaultResultState);

    useEffect(() => {
        setResult(defaultResultState);

        fetchNotifierIntegrations()
            .then((data) => {
                setResult({ notifiers: data || null, error: null, isLoading: false });
            })
            .catch((error) => {
                setResult({ notifiers: [], error, isLoading: false });
            });
    }, []);

    return result;
}

export default useFetchNotifiers;
