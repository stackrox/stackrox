import qs from 'qs';

import { ListDeployment } from 'types/deployment.proto';
import { SearchFilter, ApiSortOption } from 'types/search';
import {
    Collection,
    ResourceSelector,
    SelectorEntityType,
    SelectorField,
} from 'Containers/Collections/types';
import { getListQueryParams, getRequestQueryStringForSearchFilter } from 'utils/searchUtils';
import axios from './instance';
import { CancellableRequest, makeCancellableAxiosRequest } from './cancellationUtils';
import { Empty, Pagination } from './types';

export const collectionsBaseUrl = '/v1/collections';
export const collectionsCountUrl = '/v1/collections/count';
export const collectionsDryRunUrl = '/v1/collections/dryrun';
export const collectionsAutocompleteUrl = '/v1/collections/autocomplete';

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
    page: number,
    pageSize: number
): CancellableRequest<CollectionResponse[]> {
    const params = getListQueryParams(searchFilter, sortOption, page, pageSize);
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
            .get<{ count: number }>(`${collectionsCountUrl}?query=${query}`, { signal })
            .then((response) => response.data.count)
    );
}

export type ResolvedCollectionResponse = {
    collection: CollectionResponse;
    deployments: ListDeployment[];
};

const fieldToEntityMap: Record<SelectorField, SelectorEntityType> = {
    Deployment: 'Deployment',
    'Deployment Label': 'Deployment',
    'Deployment Annotation': 'Deployment',
    Namespace: 'Namespace',
    'Namespace Label': 'Namespace',
    'Namespace Annotation': 'Namespace',
    Cluster: 'Cluster',
    'Cluster Label': 'Cluster',
    'Cluster Annotation': 'Cluster',
};

function isAggregateError(value: Collection | AggregateError): value is AggregateError {
    return value instanceof AggregateError;
}

/**
 * This function takes a raw `CollectionResponse` from the server and parses it into a representation
 * of a `Collection` that can be supported by the current UI controls. If any incompatibilities are detected
 * it will return a list of validation errors to the caller.
 */
function parseCollection(data: CollectionResponse): Collection | AggregateError {
    const collection: Collection = {
        name: data.name,
        description: data.description,
        inUse: data.inUse,
        embeddedCollectionIds: data.embeddedCollections.map(({ id }) => id),
        selectorRules: {
            Deployment: {},
            Namespace: {},
            Cluster: {},
        },
    };

    const errors: string[] = [];

    if (data.resourceSelectors.length > 1) {
        errors.push(
            `Multiple 'ResourceSelectors' were found for this collection. Only a single resource selector is supported in the UI. Further validation errors will only apply to the first resource selector in the response.`
        );
    }

    data.resourceSelectors[0]?.rules.forEach((rule) => {
        const entity = fieldToEntityMap[rule.fieldName];
        const field = rule.fieldName;
        collection.selectorRules[entity] = collection.selectorRules[entity] ?? {
            field,
            rules: [],
        };

        const hasMultipleFieldsForEntity = collection.selectorRules[entity].field !== field;
        const isRuleForAnnotationField = rule.fieldName.endsWith('Annotation');
        const isUnsupportedRuleOperator = rule.operator !== 'OR';

        if (hasMultipleFieldsForEntity) {
            errors.push(
                `Each entity type can only contain rules for a single field. A new rule was found for [${entity} -> ${field}], when rules have already been applied for [${entity} -> ${collection.selectorRules[entity].field}].`
            );
        }
        if (isRuleForAnnotationField) {
            errors.push(
                `Collection rules for 'Annotation' field names are not supported at this time. Found field name [${rule.fieldName}].`
            );
        }
        if (isUnsupportedRuleOperator) {
            errors.push(
                `Only the disjunction operation ('OR') is currently supported in the front end collection editor. Received an operator of [${rule.operator}].`
            );
        }

        if (
            !hasMultipleFieldsForEntity &&
            !isRuleForAnnotationField &&
            !isUnsupportedRuleOperator
        ) {
            collection.selectorRules[entity].rules.push(rule);
        }
    });

    return errors.length > 0 ? new AggregateError(errors) : collection;
}

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
): CancellableRequest<Collection> {
    const params = qs.stringify(options);
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<ResolvedCollectionResponse>(`${collectionsBaseUrl}/${id}?${params}`, { signal })
            .then((response) => {
                const parsedCollection = parseCollection(response.data.collection);
                if (isAggregateError(parsedCollection)) {
                    return Promise.reject(parsedCollection);
                }
                return parsedCollection;
            })
    );
}

export type GetCollectionSelectorsResponse = {
    selectors: SelectorField[];
};

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
): CancellableRequest<CollectionResponse> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .post<CollectionResponse>(collectionsBaseUrl, collection, { signal })
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
): CancellableRequest<CollectionResponse> {
    return makeCancellableAxiosRequest((signal) =>
        axios
            .put<CollectionResponse>(`${collectionsBaseUrl}/${id}`, collection, { signal })
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
    searchCategory: SelectorField,
    searchLabel: string
): CancellableRequest<string[]> {
    const params = qs.stringify(
        { resourceSelectors, searchCategory, searchLabel },
        { arrayFormat: 'repeat' }
    );
    return makeCancellableAxiosRequest((signal) =>
        axios
            .get<{ values: string[] }>(`${collectionsAutocompleteUrl}?${params}`, { signal })
            .then((response) => response.data.values)
    );
}
