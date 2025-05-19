import { useCallback } from 'react';

import useRestQuery from 'hooks/useRestQuery';
import useURLPagination from 'hooks/useURLPagination';
import { getNetworkBaselineExternalStatus } from 'services/NetworkService';
import { NetworkBaselineExternalStatusResponse } from 'types/networkBaseline.proto';
import { getTableUIState } from 'utils/getTableUIState';

export function useNetworkBaselineStatus(
    deploymentId: string,
    status: 'ANOMALOUS' | 'BASELINE',
    initialPerPage = 10
) {
    const pagination = useURLPagination(initialPerPage, status.toLowerCase());
    const { page, perPage } = pagination;

    const fetch = useCallback(
        (): Promise<NetworkBaselineExternalStatusResponse> =>
            getNetworkBaselineExternalStatus(deploymentId, {
                page,
                perPage,
                sortOption: {},
                searchFilter: {},
            }),
        [deploymentId, page, perPage]
    );

    const { data, isLoading, error } = useRestQuery(fetch);

    const tableState = getTableUIState({
        isLoading,
        data: status === 'ANOMALOUS' ? data?.anomalous : data?.baseline,
        error,
        searchFilter: {},
    });

    const flows = status === 'ANOMALOUS' ? (data?.anomalous ?? []) : (data?.baseline ?? []);
    const total = status === 'ANOMALOUS' ? (data?.totalAnomalous ?? 0) : (data?.totalBaseline ?? 0);

    return { flows, total, tableState, pagination };
}
