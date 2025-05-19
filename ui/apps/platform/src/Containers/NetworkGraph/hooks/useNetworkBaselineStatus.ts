import { useCallback } from 'react';

import { TimeWindow } from 'constants/timeWindows';
import useRestQuery from 'hooks/useRestQuery';
import { getNetworkBaselineExternalStatus } from 'services/NetworkService';
import { NetworkBaselineExternalStatusResponse } from 'types/networkBaseline.proto';
import { getTableUIState } from 'utils/getTableUIState';

import { timeWindowToISO } from '../utils/timeWindow';

export function useNetworkBaselineStatus(
    deploymentId: string,
    timeWindow: TimeWindow,
    status: 'ANOMALOUS' | 'BASELINE'
) {
    const fetch = useCallback((): Promise<NetworkBaselineExternalStatusResponse> => {
        const fromTimestamp = timeWindowToISO(timeWindow);
        return getNetworkBaselineExternalStatus(deploymentId, fromTimestamp, {
            page: 1,
            perPage: 1000,
            sortOption: {},
            searchFilter: {},
        });
    }, [deploymentId, timeWindow]);

    const { data, isLoading, error, refetch } = useRestQuery(fetch);

    const tableState = getTableUIState({
        isLoading,
        data: status === 'ANOMALOUS' ? data?.anomalous : data?.baseline,
        error,
        searchFilter: {},
    });

    const flows = status === 'ANOMALOUS' ? (data?.anomalous ?? []) : (data?.baseline ?? []);
    const total = status === 'ANOMALOUS' ? (data?.totalAnomalous ?? 0) : (data?.totalBaseline ?? 0);

    return { flows, total, tableState, refetch };
}
