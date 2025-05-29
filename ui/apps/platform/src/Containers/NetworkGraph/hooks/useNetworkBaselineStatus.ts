import { useCallback } from 'react';

import { TimeWindow } from 'constants/timeWindows';
import useRestQuery from 'hooks/useRestQuery';
import { UseURLPaginationResult } from 'hooks/useURLPagination';
import { getNetworkBaselineExternalStatus } from 'services/NetworkService';
import { NetworkBaselineExternalStatusResponse } from 'types/networkBaseline.proto';
import { SearchFilter } from 'types/search';
import { getTableUIState } from 'utils/getTableUIState';

import { BaselineStatusType } from '../types/flow.type';
import { timeWindowToISO } from '../utils/timeWindow';

export function useNetworkBaselineStatus(
    deploymentId: string,
    timeWindow: TimeWindow,
    urlPagination: UseURLPaginationResult,
    searchFilter: SearchFilter,
    status: BaselineStatusType
) {
    const { page, perPage } = urlPagination;

    const fetch = useCallback((): Promise<NetworkBaselineExternalStatusResponse> => {
        const fromTimestamp = timeWindowToISO(timeWindow);
        return getNetworkBaselineExternalStatus(deploymentId, fromTimestamp, {
            page,
            perPage,
            sortOption: {},
            searchFilter,
        });
    }, [deploymentId, page, perPage, searchFilter, timeWindow]);

    const { data, isLoading, error, refetch } = useRestQuery(fetch);

    const tableState = getTableUIState({
        isLoading,
        data: status === 'ANOMALOUS' ? data?.anomalous : data?.baseline,
        error,
        searchFilter,
    });

    const flows = status === 'ANOMALOUS' ? (data?.anomalous ?? []) : (data?.baseline ?? []);
    const total = status === 'ANOMALOUS' ? (data?.totalAnomalous ?? 0) : (data?.totalBaseline ?? 0);

    return { flows, total, tableState, urlPagination, refetch };
}
