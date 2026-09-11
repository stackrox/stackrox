import { useCallback } from 'react';

import { fetchAlerts } from 'services/AlertsService';
import type { SearchFilter } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import useRestQuery from 'hooks/useRestQuery';

export default function useMostRecentAlerts(alertSearchFilter: SearchFilter, perPage = 3) {
    const query = getRequestQueryStringForSearchFilter(alertSearchFilter);
    const restQuery = useCallback(
        () =>
            fetchAlerts({
                alertSearchFilter,
                sortOption: { field: 'Violation Time', reversed: true },
                page: 1,
                perPage,
            }),
        // alertSearchFilter is represented by `query`, which is what the REST API uses.
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [query, perPage]
    );

    return useRestQuery(restQuery);
}
