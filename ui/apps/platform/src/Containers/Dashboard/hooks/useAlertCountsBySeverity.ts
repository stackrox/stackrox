import { useCallback } from 'react';

import { fetchSummaryAlertCounts } from 'services/AlertsService';
import { severities } from 'constants/severities';
import useRestQuery from 'hooks/useRestQuery';

export type AlertCounts = {
    [severities.LOW_SEVERITY]: number;
    [severities.MEDIUM_SEVERITY]: number;
    [severities.HIGH_SEVERITY]: number;
    [severities.CRITICAL_SEVERITY]: number;
};

function emptyAlertCounts(): AlertCounts {
    return {
        [severities.LOW_SEVERITY]: 0,
        [severities.MEDIUM_SEVERITY]: 0,
        [severities.HIGH_SEVERITY]: 0,
        [severities.CRITICAL_SEVERITY]: 0,
    };
}

export default function useAlertCountsBySeverity(query: string) {
    const restQuery = useCallback(() => {
        const { request, cancel } = fetchSummaryAlertCounts({ 'request.query': query });
        return {
            request: request.then((groups) => {
                const counts = emptyAlertCounts();
                groups[0]?.counts.forEach(({ severity, count }) => {
                    if (severity in counts) {
                        counts[severity] = Number(count);
                    }
                });
                return counts;
            }),
            cancel,
        };
    }, [query]);

    return useRestQuery(restQuery);
}
