import { useQuery } from '@apollo/client';
import type { QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type useURLPagination from 'hooks/useURLPagination';
import type { ApiSortOption } from 'types/search';
import type { VulnerabilityState } from 'types/cve.proto';

import { cveListQuery, simplifiedCveListQuery } from '../Tables/WorkloadCVEOverviewTable';
import type { ImageCVE } from '../Tables/WorkloadCVEOverviewTable';
import { getStatusesForExceptionCount } from '../../utils/searchUtils';

export function useImageCves({
    query,
    vulnerabilityState,
    pagination,
    sortOption,
    useUnifiedView = false,
    options = {},
}: {
    query: string;
    vulnerabilityState: VulnerabilityState;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption | undefined;
    useUnifiedView?: boolean;
    options?: Omit<QueryHookOptions<{ imageCVEs: ImageCVE[] }>, 'variables'>;
}) {
    const { page, perPage } = pagination;
    const activeQuery = useUnifiedView ? simplifiedCveListQuery : cveListQuery;

    return useQuery<{
        imageCVEs: ImageCVE[];
    }>(activeQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(vulnerabilityState),
        },
        ...options,
    });
}
