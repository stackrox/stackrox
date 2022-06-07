import { useEffect, useState } from 'react';
import { sortBy } from 'lodash';

import {
    AlertGroup,
    AlertQueryGroupBy,
    fetchSummaryAlertCounts,
    Severity,
} from 'services/AlertsService';

function pluckSeverityCount(severity: Severity): (group: AlertGroup) => number {
    return ({ counts }) => {
        const severityCount = counts.find((ct) => ct.severity === severity)?.count || '0';
        return -parseInt(severityCount, 10);
    };
}

export type UseViolationCountsReturn = {
    violationCounts: AlertGroup[];
    loading: boolean;
    error: Error | null;
};

export default function useViolationCounts(
    groupBy: AlertQueryGroupBy,
    query: string,
    limit = 5
): UseViolationCountsReturn {
    const [violationCounts, setViolationCounts] = useState<AlertGroup[]>([]);
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
                const selected = sortBy(groups, [
                    pluckSeverityCount('CRITICAL_SEVERITY'),
                    pluckSeverityCount('HIGH_SEVERITY'),
                    pluckSeverityCount('MEDIUM_SEVERITY'),
                    pluckSeverityCount('LOW_SEVERITY'),
                ])
                    // TODO Note: the backend does not appear to support the documented pagination queries, so we
                    // need to limit the returned number of results client side
                    .slice(0, limit)
                    // We reverse here, because PF/Victory charts stack the bars from bottom->up
                    .reverse();
                setViolationCounts(selected);
                setLoading(false);
                setError(null);
            })
            .catch((err) => {
                setLoading(true);
                setError(err);
            });

        return cancel;
    }, [groupBy, query, limit]);

    return { violationCounts, loading, error };
}
