import { useCallback } from 'react';

import type { FetchReportHistoryServiceParams } from 'services/ReportsService';
import type { ConfiguredReportSnapshot, ReportSnapshot } from 'services/ReportsService.types';
import useInterval from 'hooks/useInterval';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import useRestQuery from 'hooks/useRestQuery';

// Only the config id is needed, so accept a minimal shape to support any report type.
type ReportConfigurationRef = { id: string };

type FetchReportHistory = (
    params: FetchReportHistoryServiceParams
) => Promise<ConfiguredReportSnapshot[]>;

type ReportSnapshotLookup = Partial<Record<string, ReportSnapshot | null>>;

type Result = {
    reportSnapshots: ReportSnapshotLookup;
    isLoading: boolean;
    error: string | null;
};

export type FetchLastSnapshotReturn = Result & {
    fetchSnapshots: () => void;
};

export function useWatchLastSnapshotForReports(
    reportConfigurations: ReportConfigurationRef | ReportConfigurationRef[] | null,
    fetchReportHistory: FetchReportHistory
): FetchLastSnapshotReturn {
    const fetchSnapshotsCallback = useCallback(() => {
        if (!reportConfigurations) {
            const result: ReportSnapshotLookup = {};
            return Promise.resolve(result);
        }

        const fetchLastReportJobForConfiguration = (id: string) =>
            fetchReportHistory({
                id,
                query: '',
                page: 1,
                perPage: 1,
                showMyHistory: true,
                sortOption: {
                    field: 'Report Completion Time',
                    reversed: true,
                },
            }).then((snapshots) => snapshots[0] ?? null);

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
    }, [reportConfigurations, fetchReportHistory]);
    const { data, isLoading, error, refetch } = useRestQuery(fetchSnapshotsCallback);

    useInterval(refetch, 10000);

    const result: FetchLastSnapshotReturn = {
        reportSnapshots: data ?? {},
        isLoading,
        error: error ? getAxiosErrorMessage(error) : null,
        fetchSnapshots: refetch,
    };

    return result;
}
