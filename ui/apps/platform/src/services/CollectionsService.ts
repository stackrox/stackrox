import qs from 'qs';

import { ListDeployment } from 'types/deployment.proto';
import { SearchFilter, ApiSortOption } from 'types/search';
import { getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import axios from './instance';
import { Empty, FilterQuery } from './types';

export const collectionsBaseUrl = '/v1/collections';
export const collectionsCountUrl = '/v1/collectionscount';
export const collectionsDryRunUrl = '/v1/collections/dryrun';

export type SelectorRule = {
    fieldName: string;
    values: { value: string; matchType: string }[];
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

export type Collection = {
    id: string;
    name: string;
    description: string;
    resourceSelectors: ResourceSelector[];
    embeddedCollections: { id: string }[];
};

export type CollectionSlim = Pick<Collection, 'id' | 'name' | 'description'>;

/**
 * Fetch a paginated list of Collection objects
 */
export function listCollections(
    searchFilter: SearchFilter,
    sortOption: ApiSortOption,
    page?: number,
    pageSize?: number
): CancellableRequest<Collection[]> {
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
                collections: Collection[];
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

export type CollectionResponse = {
    collection: Collection;
};

export type CollectionResponseWithMatches = CollectionResponse & {
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
): CancellableRequest<CollectionResponseWithMatches> {
    const params = qs.stringify(options);
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<CollectionResponseWithMatches>(`${collectionsBaseUrl}/${id}?${params}`, {
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
export function createCollection(collection: CollectionRequest): CancellableRequest<Collection> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<CollectionResponse>(collectionsBaseUrl, collection, { signal })
            .then((response) => response.data.collection)
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
): CancellableRequest<Collection> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .patch<CollectionResponse>(`${collectionsBaseUrl}/${id}`, collection, {
                signal,
            })
            .then((response) => response.data.collection)
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
        filterQuery: FilterQuery;
        withMatches: boolean;
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
 * @param dryRunRequest.options.filterQuery.query
 *      A search query used to filter matching deployments
 * @param dryRunRequest.options.filterQuery.pagination
 *      Pagination options for the dry run deployment results
 * @param dryRunRequest.options.withMatches
 *      This flag will skip the resolution of matching deployments on the back end
 *      in order to do more efficient error checking. Used in order to determine if
 *      a config is valid, without returning a full data payload.
 *
 * @returns A list of deployments that are resolved by the collection.
 */
export function dryRunCollection(
    collectionRequest: CollectionRequest,
    searchFilter: SearchFilter,
    page: number,
    pageSize: number,
    sortOption?: ApiSortOption
): CancellableRequest<ListDeployment[]> {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    const dryRunRequest: CollectionDryRunRequest = {
        ...collectionRequest,
        options: {
            filterQuery: {
                query,
                pagination: {
                    offset: page * pageSize,
                    limit: pageSize,
                    ...(sortOption && { sortOption }),
                },
            },
            withMatches: true,
        },
    };
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<CollectionDryRunResponse>(collectionsDryRunUrl, dryRunRequest, {
                signal,
            })
            .then((response) => response.data.deployments)
    );
}
