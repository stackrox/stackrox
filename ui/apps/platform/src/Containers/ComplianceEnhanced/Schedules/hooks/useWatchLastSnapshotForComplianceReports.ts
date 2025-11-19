import { useCallback } from 'react';

import useInterval from 'hooks/useInterval';
import { getAxiosErrorMessage } from 'utils/responseErrorUtils';
import { fetchComplianceReportHistory } from 'services/ComplianceScanConfigurationService';
import type {
    ComplianceReportSnapshot,
    ComplianceScanConfigurationStatus,
} from 'services/ComplianceScanConfigurationService';
import useRestQuery from 'hooks/useRestQuery';

async function fetchLastJobForConfiguration(
    scanConfigId: string
): Promise<ComplianceReportSnapshot | null> {
    const complianceReportSnapshots = await fetchComplianceReportHistory({
        id: scanConfigId,
        query: '',
        page: 1,
        perPage: 1,
        showMyHistory: true,
        sortOption: {
            field: 'Compliance Report Completed Time',
            reversed: true,
        },
    });

    return complianceReportSnapshots[0] ?? null;
}

type ComplianceReportSnapshotLookup = Partial<Record<string, ComplianceReportSnapshot | null>>;

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
    scanConfigurations:
        | ComplianceScanConfigurationStatus
        | ComplianceScanConfigurationStatus[]
        | undefined
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
                        (acc, { id }, index) => {
                            const result = snapshotResults[index];
                            if (result.status === 'fulfilled') {
                                return { ...acc, [id]: result.value };
                            }
                            return { ...acc, [id]: null };
                        },
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
