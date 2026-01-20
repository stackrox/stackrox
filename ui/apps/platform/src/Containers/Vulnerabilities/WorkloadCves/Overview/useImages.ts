import { useQuery } from '@apollo/client';
import type { QueryHookOptions } from '@apollo/client';

import { getPaginationParams } from 'utils/searchUtils';
import type { ApiSortOption } from 'types/search';
import type useURLPagination from 'hooks/useURLPagination';
import useFeatureFlags from 'hooks/useFeatureFlags';
import { imageListQuery, imageV2ListQuery } from '../Tables/ImageOverviewTable';
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
    const { isFeatureFlagEnabled } = useFeatureFlags();
    const isNewImageDataModelEnabled = isFeatureFlagEnabled('ROX_FLATTEN_IMAGE_DATA');
    const { page, perPage } = pagination;
    return useQuery<{
        images: Image[];
    }>(isNewImageDataModelEnabled ? imageV2ListQuery : imageListQuery, {
        variables: {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        ...options,
    });
}
