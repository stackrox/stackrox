import { useCallback } from 'react';
import { fetchImagesCount } from 'services/imageService';
import useRestQuery from 'hooks/useRestQuery';

/**
 * Fetch image count via GET /v1/imagescount.
 * `query` is the same RawQuery string used by GraphQL imageCount.
 */
export default function useFetchImageCount(query = '') {
    const restQuery = useCallback(() => fetchImagesCount(query), [query]);

    return useRestQuery(restQuery);
}
