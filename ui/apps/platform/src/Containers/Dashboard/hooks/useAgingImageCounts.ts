import { useCallback } from 'react';

import { fetchImageCount } from 'services/imageService';
import useRestQuery from 'hooks/useRestQuery';

import type { TimeRangeCounts } from '../Widgets/AgingImagesChart';

export default function useAgingImageCounts(queries: [string, string, string, string]) {
    const [query0, query1, query2, query3] = queries;
    const restQuery = useCallback(
        () =>
            Promise.all([
                fetchImageCount(query0),
                fetchImageCount(query1),
                fetchImageCount(query2),
                fetchImageCount(query3),
            ]).then(
                ([timeRange0, timeRange1, timeRange2, timeRange3]): TimeRangeCounts => ({
                    timeRange0,
                    timeRange1,
                    timeRange2,
                    timeRange3,
                })
            ),
        [query0, query1, query2, query3]
    );

    return useRestQuery(restQuery);
}
