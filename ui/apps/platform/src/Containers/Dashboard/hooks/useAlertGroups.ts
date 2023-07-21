import { useCallback } from 'react';
import { AlertQueryGroupBy, fetchSummaryAlertCounts } from 'services/AlertsService';
import useRestQuery from 'hooks/useRestQuery';

export default function useAlertGroups(query: string, groupBy?: AlertQueryGroupBy) {
    const restQuery = useCallback(
        () => fetchSummaryAlertCounts({ 'request.query': query, group_by: groupBy }),
        [groupBy, query]
    );

    return useRestQuery(restQuery);
}
