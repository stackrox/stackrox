import { useEffect, useState } from 'react';

import { AlertGroup, AlertQueryGroupBy, fetchSummaryAlertCounts } from 'services/AlertsService';

export type UseAlertGroupsReturn = {
    alertGroups: AlertGroup[];
    loading: boolean;
    error: Error | null;
};

export default function useAlertGroups(
    groupBy: AlertQueryGroupBy,
    query: string
): UseAlertGroupsReturn {
    const [alertGroups, setAlertGroups] = useState<AlertGroup[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<Error | null>(null);

    useEffect(() => {
        const { request, cancel } = fetchSummaryAlertCounts({
            'request.query': query,
            group_by: groupBy,
        });

        setError(null);

        request
            .then((groups) => {
                setAlertGroups(groups);
                setLoading(false);
                setError(null);
            })
            .catch((err) => {
                setLoading(true);
                setError(err);
            });

        return cancel;
    }, [groupBy, query]);

    return { alertGroups, loading, error };
}
