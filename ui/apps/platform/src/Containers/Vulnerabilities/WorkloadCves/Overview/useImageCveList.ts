import { useQuery } from '@apollo/client';
import type { QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import { getStatusesForExceptionCount } from 'Containers/Vulnerabilities/utils/searchUtils';
import type useURLPagination from 'hooks/useURLPagination';
import type { ApiSortOption } from 'types/search';
import type { VulnerabilityState } from 'types/cve.proto';

import { cveListQuery } from '../Tables/WorkloadCVEOverviewTable';
import type { ImageCVE } from '../Tables/WorkloadCVEOverviewTable';

export function useImageCveList({
    query,
    vulnerabilityState,
    pagination,
    sortOption,
    options = {},
}: {
    query: string;
    vulnerabilityState: VulnerabilityState;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption | undefined;
    options?: Omit<QueryHookOptions<{ imageCVEs: ImageCVE[] }>, 'variables'>;
}) {
    const { page, perPage } = pagination;

    return useQuery<{
        imageCVEs: ImageCVE[];
    }>(cveListQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(vulnerabilityState),
        },
        ...options,
    });
}
