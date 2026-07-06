import queryString from 'qs';

import type { ListSecret } from 'types/secret.proto';
import type { SearchFilter, SearchQueryOptions } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
// TODO: Move buildNestedRawQueryParams to searchUtils as a shared helper
import { buildNestedRawQueryParams } from './ComplianceCommon';
import axios from './instance';
import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';

const baseUrl = '/v1/secretsextended';
const baseCountUrl = '/v1/secretscount';

export function fetchSecrets({
    searchFilter,
    sortOption,
    page,
    perPage,
}: SearchQueryOptions): CancellableRequest<ListSecret[]> {
    const params = buildNestedRawQueryParams({ searchFilter, sortOption, page, perPage });
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ secrets: ListSecret[] }>(`${baseUrl}?${params}`, { signal })
            .then((response) => response?.data?.secrets ?? [])
    );
}

export function fetchSecretCount(searchFilter: SearchFilter): CancellableRequest<number> {
    const params = queryString.stringify(
        { query: getRequestQueryStringForSearchFilter(searchFilter) },
        { arrayFormat: 'repeat' }
    );
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ count: number }>(`${baseCountUrl}?${params}`, { signal })
            .then((response) => response?.data?.count ?? 0)
    );
}
