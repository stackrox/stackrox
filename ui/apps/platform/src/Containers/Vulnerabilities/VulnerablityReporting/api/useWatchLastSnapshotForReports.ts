import { useCallback, useEffect, useState } from 'react';

import { fetchReportHistory } from 'services/ReportsService';
import { ReportConfiguration, ReportSnapshot } from 'services/ReportsService.types';
import useInterval from 'hooks/useInterval';
import { getRequestQueryString } from './apiUtils';
import { getErrorMessage } from '../errorUtils';

type Result = {
    reportSnapshots: Record<string, ReportSnapshot>;
    isLoading: boolean;
    error: string | null;
};

export type FetchLastSnapshotReturn = Result & {
    fetchSnapshots: () => void;
};

async function fetchLastReportJobForConfiguration(
    reportConfigurationId: string
): Promise<ReportSnapshot> {
    // Query for the current user's last report job
    const query = getRequestQueryString({ 'Report state': ['PREPARING', 'WAITING'] });

    const reportSnapshot = await fetchReportHistory({
        id: reportConfigurationId,
        query,
        page: 1,
        perPage: 1,
        showMyHistory: true,
        sortOption: {
            field: 'Report Completion Time',
            reversed: true,
        },
    });
    return reportSnapshot[0] ?? null;
}

const defaultResult = {
    reportSnapshots: {},
    isLoading: false,
    error: null,
};

export function useWatchLastSnapshotForReports(
    reportConfigurations: ReportConfiguration | ReportConfiguration[] | null
): FetchLastSnapshotReturn {
    const [result, setResult] = useState<Result>(defaultResult);

    const fetchSnapshots = useCallback(async () => {
        if (!reportConfigurations) {
            return;
        }

        setResult((prevResult) => ({
            ...prevResult,
            isLoading: true,
            error: null,
        }));

        const configurations = Array.isArray(reportConfigurations)
            ? reportConfigurations
            : [reportConfigurations];

        try {
            const snapshots = await Promise.all(
                configurations.map(({ id }) => fetchLastReportJobForConfiguration(id))
            );
            setResult({
                reportSnapshots: configurations.reduce(
                    (acc, { id }, index) => ({ ...acc, [id]: snapshots[index] }),
                    {}
                ),
                isLoading: false,
                error: null,
            });
        } catch (error) {
            setResult({
                reportSnapshots: {},
                isLoading: false,
                error: getErrorMessage(error),
            });
        }
    }, [reportConfigurations]);

    useInterval(fetchSnapshots, 10000);

    useEffect(() => {
        // eslint-disable-next-line no-void
        void fetchSnapshots();
        // Clear out statuses when report configurations change to avoid stale renders across pages
        setResult(defaultResult);
    }, [fetchSnapshots]);

    return {
        ...result,
        fetchSnapshots,
    };
}
