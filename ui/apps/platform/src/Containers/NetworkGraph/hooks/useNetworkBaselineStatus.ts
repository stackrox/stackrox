import { useCallback, useEffect } from 'react';

import { TimeWindow } from 'constants/timeWindows';
import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import { getNetworkBaselineExternalStatus } from 'services/NetworkService';
import { NetworkBaselineExternalStatusResponse } from 'types/networkBaseline.proto';
import { getTableUIState } from 'utils/getTableUIState';

import { DEFAULT_NETWORK_GRAPH_PAGE_SIZE } from '../NetworkGraph.constants';
import { timeWindowToISO } from '../utils/timeWindow';

export function useNetworkBaselineStatus(
    deploymentId: string,
    timeWindow: TimeWindow,
    status: 'ANOMALOUS' | 'BASELINE'
) {
    const pagination = useURLPagination(DEFAULT_NETWORK_GRAPH_PAGE_SIZE, status.toLowerCase());
    const { page, perPage, setPage } = pagination;

    const fetch = useCallback((): Promise<NetworkBaselineExternalStatusResponse> => {
        const fromTimestamp = timeWindowToISO(timeWindow);
        return getNetworkBaselineExternalStatus(deploymentId, fromTimestamp, {
            page,
            perPage,
            sortOption: {},
            searchFilter: {},
        });
    }, [deploymentId, page, perPage, timeWindow]);

    const { data, isLoading, error, refetch } = useRestQuery(fetch);

    useEffect(() => {
        setPage(1);
    }, [deploymentId, timeWindow, setPage]);

    const tableState = getTableUIState({
        isLoading,
        data: status === 'ANOMALOUS' ? data?.anomalous : data?.baseline,
        error,
        searchFilter: {},
    });

    const flows = status === 'ANOMALOUS' ? (data?.anomalous ?? []) : (data?.baseline ?? []);
    const total = status === 'ANOMALOUS' ? (data?.totalAnomalous ?? 0) : (data?.totalBaseline ?? 0);

    return { flows, total, tableState, pagination, refetch };
}
