import { useQuery } from '@apollo/client';
import type { QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type { ApiSortOption } from 'types/search';
import type useURLPagination from 'hooks/useURLPagination';
import { imageV2ListQuery } from '../Tables/ImageOverviewTable';
import type { Image } from '../Tables/ImageOverviewTable';

export function useImages({
    query,
    pagination,
    sortOption,
    options = {},
}: {
    query: string;
    pagination: ReturnType<typeof useURLPagination>;
    sortOption: ApiSortOption | undefined;
    options?: Omit<QueryHookOptions<{ images: Image[] }>, 'variables'>;
}) {
    const { page, perPage } = pagination;
    return useQuery<{
        images: Image[];
    }>(imageV2ListQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        ...options,
    });
}
