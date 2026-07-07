import queryString from 'qs';

import type { SearchFilter, SearchQueryOptions } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
// TODO: Move buildNestedRawQueryParams to searchUtils as a shared helper
import { buildNestedRawQueryParams } from './ComplianceCommon';
import axios from './instance';
import { makeCancellableAxiosRequest } from './cancellationUtils';
import type { CancellableRequest } from './cancellationUtils';

const baseUrl = '/v1/secretsextended';
const baseCountUrl = '/v1/secretscount';

export const secretTypes = [
    'UNDETERMINED',
    'PUBLIC_CERTIFICATE',
    'CERTIFICATE_REQUEST',
    'PRIVACY_ENHANCED_MESSAGE',
    'OPENSSH_PRIVATE_KEY',
    'PGP_PRIVATE_KEY',
    'EC_PRIVATE_KEY',
    'RSA_PRIVATE_KEY',
    'DSA_PRIVATE_KEY',
    'CERT_PRIVATE_KEY',
    'ENCRYPTED_PRIVATE_KEY',
    'IMAGE_PULL_SECRET',
] as const;

export type SecretType = (typeof secretTypes)[number];

export type ListSecret = {
    id: string;
    name: string;
    clusterId: string;
    clusterName: string;
    namespace: string;
    types: SecretType[];
    createdAt: string; // ISO 8601 string
};

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
