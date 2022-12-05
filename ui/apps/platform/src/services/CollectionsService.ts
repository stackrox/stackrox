import qs from 'qs';

import { ListDeployment } from 'types/deployment.proto';
import { SearchFilter, ApiSortOption } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import axios from './instance';
import { Empty, Pagination } from './types';

export const collectionsBaseUrl = '/v1/collections';
export const collectionsCountUrl = '/v1/collectionscount';
export const collectionsDryRunUrl = '/v1/collections/dryrun';
export const collectionsAutocompleteUrl = '/v1/collections/autocomplete';

export type SelectorRule = {
    fieldName: string;
    values: { value: string }[];
    operator: 'AND' | 'OR';
};

export type ResourceSelector = {
    rules: SelectorRule[];
};

export type CollectionRequest = {
    name: string;
    description: string;
    resourceSelectors: ResourceSelector[];
    embeddedCollectionIds: string[];
};

export type CollectionResponse = {
    id: string;
    name: string;
    description: string;
    inUse: boolean;
    resourceSelectors: ResourceSelector[];
    embeddedCollections: { id: string }[];
};

/**
 * Fetch a paginated list of Collection objects
 */
export function listCollections(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): CancellableRequest<CollectionResponse[]> {
    let offset: number | undefined;
    if (typeof page === 'number' && typeof pageSize === 'number') {
        offset = page > 0 ? page * pageSize : 0;
    }
    const query = {
        query: getRequestQueryStringForSearchFilter(searchFilter),
        pagination: { offset, limit: pageSize, sortOption },
    };
    const params = qs.stringify({ query }, { allowDots: true });
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{
                collections: CollectionResponse[];
            }>(`${collectionsBaseUrl}?${params}`, { signal })
            .then((response) => response.data.collections)
    );
}

/**
 * Fetch the number of collections
 */
export function getCollectionCount(searchFilter: SearchFilter): CancellableRequest<number> {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ count: number }>(`${collectionsCountUrl}?query.query=${query}`, { signal })
            .then((response) => response.data.count)
    );
}

export type ResolvedCollectionResponse = {
    collection: CollectionResponse;
};

export type ResolvedCollectionResponseWithMatches = ResolvedCollectionResponse & {
    deployments: ListDeployment[];
};

/**
 * Fetch a single collection by id
 *
 * @param id
 *      The collection ID
 * @param options.withMatches
 *      When true, returns the list of deployments that match the collection
 *      rules, otherwise returns an empty array.
 */
export function getCollection(
    id: string,
    options: { withMatches: boolean } = { withMatches: false }
): CancellableRequest<ResolvedCollectionResponseWithMatches> {
    const params = qs.stringify(options);
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<ResolvedCollectionResponseWithMatches>(`${collectionsBaseUrl}/${id}?${params}`, {
                signal,
            })
            .then((response) => response.data)
    );
}

/**
 * Create a new collection
 *
 * @param collection
 *      The collection object details to be created
 * @returns
 *      The created collection object, with ID
 */
export function createCollection(
    collection: CollectionRequest
): CancellableRequest<ResolvedCollectionResponse> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<ResolvedCollectionResponse>(collectionsBaseUrl, collection, { signal })
            .then((response) => response.data)
    );
}

/**
 * Updates an existing collection object.
 *
 * @param id
 *      The ID of the collection to update
 * @param collection
 *      The new collection object details.
 * @returns
 *      The updated collection object
 *
 */
export function updateCollection(
    id: string,
    collection: CollectionRequest
): CancellableRequest<ResolvedCollectionResponse> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .patch<ResolvedCollectionResponse>(`${collectionsBaseUrl}/${id}`, collection, {
                signal,
            })
            .then((response) => response.data)
    );
}

/**
 * Deletes a collection
 *
 * @param id
 *      The ID of the collection to delete
 */
export function deleteCollection(id: string): CancellableRequest<Empty> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .delete<Empty>(`${collectionsBaseUrl}/${id}`, { signal })
            .then((response) => response.data)
    );
}

export type CollectionDryRunRequest = CollectionRequest & {
    options: {
        pagination: Pagination;
        skipDeploymentMatching: boolean;
    };
};

export type CollectionDryRunResponse = {
    deployments: ListDeployment[];
};

/**
 * Fetches the currently matching deployments for a collection based on the applied resource
 * selectors and embedded collections. Note that the deployments in a collection are resolved
 * dynamically and may change over time as new deployments are added to a cluster.
 *
 * @param dryRunRequest.resourceSelectors
 *      The resource selector rules used to match deployments
 * @param dryRunRequest.embeddedCollectionIds
 *      An array of collection ids whose matching deployments should
 *      be added to the result set.
 * @param dryRunRequest.options.pagination
 *      Pagination options for the dry run deployment results
 * @param dryRunRequest.options.skipDeploymentMatching
 *      This flag will skip the resolution of matching deployments on the back end
 *      in order to do more efficient error checking. Used in order to determine if
 *      a config is valid, without returning a full data payload.
 *
 * @returns A list of deployments that are resolved by the collection.
 */
export function dryRunCollection(
    dryRunRequest: CollectionDryRunRequest
    // TODO `ListDeployment` will make this impossible to paginate without loading the entire
    // dataset client side. Ask [BE] if there is an efficient way to aggregate namespaces/clusters
    // under a matching deployment name similar to the graphql query. An alternative might be to
    // change the rendering of the list to not group deployments, but to sort alphabetically.
): CancellableRequest<ListDeployment[]> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<CollectionDryRunResponse>(collectionsDryRunUrl, dryRunRequest, {
                signal,
            })
            .then((response) => response.data.deployments)
    );
}

/**
 * Function that returns a list of autocomplete suggestions for selector fields based on resources
 * that match the provided resource selectors.
 *
 * @param resourceSelectors
 *      The resource selectors used to scope the autocomplete search
 * @param searchCategory
 *      The field that autocomplete results should be returned for
 * @param searchLabel
 *      The user provided search text
 */
export function getCollectionAutoComplete(
    resourceSelectors: ResourceSelector[],
    searchCategory: string,
    searchLabel: string
): CancellableRequest<string[]> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<{ values: string[] }>(
                collectionsAutocompleteUrl,
                { resourceSelectors, searchCategory, searchLabel },
                { signal }
            )
            .then((response) => response.data.values)
    );
}
