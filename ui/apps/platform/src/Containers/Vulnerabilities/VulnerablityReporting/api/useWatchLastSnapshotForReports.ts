import { useCallback } from 'react';

import { fetchReportHistory } from 'services/ReportsService';
import { ReportConfiguration, ReportSnapshot } from 'services/ReportsService.types';
import useInterval from 'hooks/useInterval';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRestQuery from 'hooks/useRestQuery';
import { getRequestQueryString } from './apiUtils';

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

type ReportSnapshotLookup = Record<string, ReportSnapshot | null>;

type Result = {
    reportSnapshots: ReportSnapshotLookup;
    isLoading: boolean;
    error: string | null;
};

export type FetchLastSnapshotReturn = Result & {
    fetchSnapshots: () => void;
};

export function useWatchLastSnapshotForReports(
    reportConfigurations: ReportConfiguration | ReportConfiguration[] | null
): FetchLastSnapshotReturn {
    const fetchSnapshotsCallback = useCallback(async () => {
        if (!reportConfigurations) {
            const result: ReportSnapshotLookup = {};
            return Promise.resolve(result);
        }

        const promise: Promise<ReportSnapshotLookup> = new Promise((resolve, reject) => {
            const configurations = Array.isArray(reportConfigurations)
                ? reportConfigurations
                : [reportConfigurations];

            Promise.all(configurations.map(({ id }) => fetchLastReportJobForConfiguration(id)))
                .then((snapshotResults) => {
                    const result: ReportSnapshotLookup = configurations.reduce(
                        (acc, { id }, index) => ({ ...acc, [id]: snapshotResults[index] }),
                        {}
                    );
                    resolve(result);
                })
                .catch((error) => {
                    reject(error);
                });
        });

        return promise;
    }, [reportConfigurations]);
    const { data, isLoading, error, refetch } = useRestQuery(fetchSnapshotsCallback);

    useInterval(refetch, 10000);

    const result: FetchLastSnapshotReturn = {
        reportSnapshots: data || {},
        isLoading,
        error: error ? getAxiosErrorMessage(error) : null,
        fetchSnapshots: refetch,
    };

    return result;
}
