import { useCallback } from 'react';

import useInterval from 'hooks/useInterval';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import {
    ComplianceReportSnapshot,
    ComplianceScanConfigurationStatus,
    fetchComplianceReportHistory,
} from 'services/ComplianceScanConfigurationService';
import useRestQuery from 'hooks/useRestQuery';

async function fetchLastJobForConfiguration(
    scanConfigId: string
): Promise<ComplianceReportSnapshot | null> {
    const query = getRequestQueryStringForSearchFilter({
        'Report state': ['PREPARING', 'WAITING'],
    });

    const complianceReportSnapshots = await fetchComplianceReportHistory({
        id: scanConfigId,
        query,
        page: 1,
        perPage: 1,
        showMyHistory: true,
        sortOption: {
            field: 'Report Completion Time',
            reversed: true,
        },
    });

    return complianceReportSnapshots[0] ?? null;
}

type ComplianceReportSnapshotLookup = Partial<Record<string, ComplianceReportSnapshot>>;

type Result = {
    complianceReportSnapshots: ComplianceReportSnapshotLookup;
    isLoading: boolean;
    error: string | null;
};

export type FetchLastComplianceReportSnapshotReturn = Result & {
    fetchSnapshots: () => void;
};

// @TODO: Handle error states better for this polling scenario
function useWatchLastSnapshotForComplianceReports(
    scanConfigurations: ComplianceScanConfigurationStatus[] | undefined
): FetchLastComplianceReportSnapshotReturn {
    const fetchSnapshotsCallback = useCallback(() => {
        if (!scanConfigurations) {
            const result: ComplianceReportSnapshotLookup = {};
            return Promise.resolve(result);
        }

        const promise: Promise<ComplianceReportSnapshotLookup> = new Promise((resolve, reject) => {
            const configurations = Array.isArray(scanConfigurations)
                ? scanConfigurations
                : [scanConfigurations];

            Promise.allSettled(configurations.map(({ id }) => fetchLastJobForConfiguration(id)))
                .then((snapshotResults) => {
                    const result: ComplianceReportSnapshotLookup = configurations.reduce(
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
    }, [scanConfigurations]);
    const { data, isLoading, error, refetch } = useRestQuery(fetchSnapshotsCallback);

    useInterval(refetch, 10000);

    const result: FetchLastComplianceReportSnapshotReturn = {
        complianceReportSnapshots: data || {},
        isLoading,
        error: error ? getAxiosErrorMessage(error) : null,
        fetchSnapshots: refetch,
    };

    return result;
}

export default useWatchLastSnapshotForComplianceReports;
