import { useQuery } from '@apollo/client';
import type { DocumentNode, QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type useURLPagination from 'hooks/useURLPagination';
import useFeatureFlags from 'hooks/useFeatureFlags';
import type { ApiSortOption } from 'types/search';
import type { VulnerabilityState } from 'types/cve.proto';

import { getCveListQuery } from '../Tables/WorkloadCVEOverviewTable';
import type { ImageCVE } from '../Tables/WorkloadCVEOverviewTable';
import { getStatusesForExceptionCount } from '../../utils/searchUtils';

export function useImageCves({
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const gqlQuery: DocumentNode = getCveListQuery(isFeatureFlagEnabled);

    return useQuery<{
        imageCVEs: ImageCVE[];
    }>(gqlQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
            statusesForExceptionCount: getStatusesForExceptionCount(vulnerabilityState),
        },
        ...options,
    });
}
