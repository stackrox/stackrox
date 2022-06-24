import { useCallback } from 'react';
import { AlertQueryGroupBy, fetchSummaryAlertCounts } from 'services/AlertsService';
import useRestQuery from './useRestQuery';

export default function useAlertGroups(groupBy: AlertQueryGroupBy, query: string) {
    const restQuery = useCallback(
        () => fetchSummaryAlertCounts({ 'request.query': query, group_by: groupBy }),
        [groupBy, query]
    );

    return useRestQuery(restQuery);
}
