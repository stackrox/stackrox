import { useReducer, useEffect, Dispatch } from 'react';
import sortBy from 'lodash/sortBy';

import { Collection, listCollections } from 'services/CollectionsService';
import { ensureExhaustive } from 'utils/type.utils';

type CollectionMap = Record<string, Collection>;

// `pageSize` is the default number of items that will attempt to be pulled when the user
// loads more items in the detached collections section
const pageSize = 10;
// `minimumUpdateSize` is the minimum number of results that would be added to the UI without
// triggering an additional request. If after client side filtering, the number of new collections
// is below this number, we will pull another page worth of data.
const minimumUpdateSize = pageSize / 2;

/**
 * Fetches more collections, aiming to update the client table with a minimum
 * number of results. Collections that are already present client side will be filtered out
 * from the server response. The number of collections added to the user's screen should be
 * `minimumClientFetchResults` <= collections.length <= `pageSize`
 *
 * If the actual results to add to the detached collections list after filtering is not over the
 * minimum, we fetch the next page and concat the two result lists. Repeat this process until
 * we have enough results to show the user.
 *
 * If the response from the server contains less collections then the page size, display all
 * aggregated results and set `hasMore` to `false`.
 * @param clientMap
 *      A mapping of {id -> collection} for all attached and detached collections that
 *      are currently displayed in the UI.
 * @param searchValue
 *      A search string to used to filter collections by name.
 * @param pageNumber
 *      The current page of collections to fetch.
 * @param aggregateResult
 *      The aggregated list of collections built up by multiple successive calls to this function.
 * @returns
 *      A Promise that resolves with the retrieved detached collection list, the next
 *      number, and the number of items returned in the last call to the server.
 */
function fetchDetachedCollections(
    excludedCollectionId: string | null,
    clientMap: CollectionMap,
    searchValue: string,
    pageNumber: number,
    aggregateResult: Collection[]
): Promise<{
    detached: Collection[];
    nextPage: number;
    lastResponseSize: number;
}> {
    const searchOption = { 'Collection Name': searchValue };
    const { request } = listCollections(
        searchOption,
        { field: 'Collection Name', reversed: false },
        pageNumber - 1,
        pageSize
    );

    return request.then((collections) => {
        const newDetached = collections.filter(
            ({ id }) => !clientMap[id] && id !== excludedCollectionId
        );
        const detached = aggregateResult.concat(newDetached);
        const lastResponseSize = collections.length;
        const shouldFetchMore =
            lastResponseSize === pageSize && detached.length < minimumUpdateSize;
        const nextPage = pageNumber + 1;

        if (shouldFetchMore) {
            return fetchDetachedCollections(
                excludedCollectionId,
                clientMap,
                searchValue,
                nextPage,
                detached
            );
        }
        return Promise.resolve({ detached, nextPage, lastResponseSize });
    });
}

/**
 * Given the current attached and detached lists displayed to the user, attempt to fetch
 * more detached collections from the server.
 */
function fetchMore(
    excludedCollectionId: string | null,
    attached: CollectionMap,
    detached: CollectionMap,
    searchValue: string,
    currentPage: number,
    dispatch: Dispatch<ReducerPayload>
) {
    dispatch({ type: 'fetchMoreRequest' });

    fetchDetachedCollections(
        excludedCollectionId,
        { ...attached, ...detached },
        searchValue,
        currentPage,
        []
    )
        .then(({ detached: newDetached, nextPage, lastResponseSize }) => {
            const nextDetached = { ...detached };
            newDetached.forEach((collection) => {
                nextDetached[collection.id] = collection;
            });
            dispatch({
                type: 'fetchMoreComplete',
                detached: nextDetached,
                hasMore: lastResponseSize >= pageSize,
                page: nextPage,
            });
        })
        .catch((error) => {
            dispatch({ type: 'fetchMoreError', error });
        });
}

function moveItem(from: CollectionMap, to: CollectionMap, id: string) {
    const toMap = { ...to };
    const fromMap = { ...from };
    const item = fromMap[id];
    if (item) {
        toMap[id] = item;
        delete fromMap[id];
    }
    return [fromMap, toMap];
}

function arrayToMap(collections: Collection[]): Record<string, Collection> {
    const map = {};
    collections.forEach(({ id, ...rest }) => {
        map[id] = { id, ...rest };
    });
    return map;
}

type UseEmbeddedCollectionsState = {
    page: number;
    hasMore: boolean;
    isFetchingMore: boolean;
    fetchMoreError: Error | null;
    attached: CollectionMap;
    detached: CollectionMap;
    search: string;
};

type ReducerPayload =
    | { type: 'fetchMoreRequest' }
    | { type: 'fetchMoreComplete'; detached: CollectionMap; page: number; hasMore: boolean }
    | { type: 'fetchMoreError'; error: Error }
    | { type: 'attachCollection'; id: string }
    | { type: 'detachCollection'; id: string }
    | { type: 'resetDetachedList'; search: string };

function embeddedCollectionsReducer(
    state: UseEmbeddedCollectionsState,
    payload: ReducerPayload
): UseEmbeddedCollectionsState {
    switch (payload.type) {
        case 'fetchMoreRequest':
            return { ...state, isFetchingMore: true };
        case 'fetchMoreComplete': {
            return { ...state, ...payload, isFetchingMore: false, fetchMoreError: null };
        }
        case 'fetchMoreError':
            return {
                ...state,
                isFetchingMore: false,
                fetchMoreError: payload.error,
            };
        case 'attachCollection': {
            const { id } = payload;
            const [detached, attached] = moveItem(state.detached, state.attached, id);
            return { ...state, attached, detached };
        }
        case 'detachCollection': {
            const { id } = payload;
            const [attached, detached] = moveItem(state.attached, state.detached, id);
            return { ...state, attached, detached };
        }
        case 'resetDetachedList':
            return { ...state, search: payload.search, page: 1, hasMore: true, detached: {} };
        default:
            return ensureExhaustive(payload);
    }
}
function byNameCaseInsensitive(collection: Collection) {
    return collection.name.toLowerCase();
}

const initialState = {
    page: 1,
    hasMore: true,
    isFetchingMore: false,
    fetchMoreError: null,
    detached: {},
    search: '',
};

export type UseEmbeddedCollectionsReturn = {
    /** Client side state of attached collections */
    attached: Collection[];
    /** Client side state of detached collections */
    detached: Collection[];
    /** Move a collection from detached -> attached by id */
    attach: (id: string) => void;
    /** Move a collection from attached -> detached by id */
    detach: (id: string) => void;
    /** Whether or not more collections might be available to be loaded from the server */
    hasMore: boolean;
    /** Loads another page of collections from the server */
    fetchMore: (search: string) => void;
    /** Callback to fire when the search string changes */
    onSearch: (search: string) => void;
    /** Whether or not the current fetchMore request is loading */
    isFetchingMore: boolean;
    /** If a fetchMore request fails, the error, or null */
    fetchMoreError: Error | null;
};

/**
 * This hook maintains the 'attached' and 'detached' list of collections that are displayed in the
 * UI and provides functions that can be used to modify them.
 *
 * Upon initialization, do the following:
 * - Load a subset (page) of detached collections and store in a local object.
 * - Store the most recent requested page number of detached collections.
 * - Store whether or not _all_ of the detached collections have been loaded from the backend.
 *
 * When the user attaches a collection, remove it from the unattached list and
 * add it to the attached list. Vice-versa when a user un-attaches a collection.
 *
 * If all of the unattached collections have not yet been loaded, display a "View more" button below the
 * unattached list. If the user clicks this button, fetch the next page of unattached collections.
 * - If there less results than the page size, remove the View more button. All items have been loaded client side.
 * - If there are > zero results, remove any items from the results that exist in the
 *   "attached" object or "detached" object to prevent rendering duplicates. This will occur when the
 *   user has loaded some items and made changes to which collections are attached without saving.
 * - If a "View more" request returns results, but the results are reduced below a threshold
 *   when filtering due to local collection state, automatically refetch the next page for better UX.
 *   This process will repeat until either no collections are returned from the server or the threshold
 *   of items to display has been met.
 *
 * When the user enters a value in the search box:
 *  - Filter the list of attached collections _during rendering only_.
 *  - Clear the current page number, "hasMore" boolean, and local cache of unattached collections.
 *  - Fetch the first page of unattached collections again, and resume the above functionality with the
 *    filter in place.
 *  - [Note] We cannot just filter the unattached list during rendering and make subsequent page fetches
 *    with the search filter applied as that would leave holes in the unfiltered cache data when the user
 *    clears the search value. On the other hand, running multiple "fetch more" requests and filtering
 *    results by the search filter could result in many useless requests that lead to a bad UX.
 *  - [Bonus] It may be nice in the future to be able to maintain multiple caches, keyed by search term, to avoid
 *    many requests if the user clears the search box or reuses the search terms. Alternatively we could just keep a
 *    "no search" cache, and a "search" cache to cover the two most expected use cases.
 *  - [Bonus] In the future, we should track when the user has loaded all collections _without_ search filtering
 *    as we can then disable the cache clearing/fetching behavior and filter the client side cache directly.
 *
 * @param excludedCollectionId
 *      The ids of the main collection that the other collections are being attached to, or `null` if
 *      a new collection is being created.
 * @param initialAttachedCollectionIds
 *      A list of attached collection ids used to populate the initial attached collection list.
 */
export default function useEmbeddedCollections(
    excludedCollectionId: string | null,
    initialAttachedCollections: Collection[]
): UseEmbeddedCollectionsReturn {
    const [state, dispatch] = useReducer(embeddedCollectionsReducer, initialState, (init) => ({
        ...init,
        attached: arrayToMap(initialAttachedCollections),
    }));

    const { attached, detached, page, hasMore, isFetchingMore, fetchMoreError, search } = state;

    useEffect(() => {
        return fetchMore(
            excludedCollectionId,
            arrayToMap(initialAttachedCollections),
            {},
            '',
            1,
            dispatch
        );
    }, [excludedCollectionId, initialAttachedCollections]);

    const onSearch = (newSearch: string) => {
        dispatch({ type: 'resetDetachedList', search: newSearch });
        fetchMore(excludedCollectionId, attached, {}, newSearch, 1, dispatch);
    };

    return {
        attached: sortBy(Object.values(attached), byNameCaseInsensitive),
        detached: sortBy(Object.values(detached), byNameCaseInsensitive),
        attach: (id: string) => dispatch({ type: 'attachCollection', id }),
        detach: (id: string) => dispatch({ type: 'detachCollection', id }),
        hasMore,
        fetchMore: () =>
            fetchMore(excludedCollectionId, attached, detached, search, page, dispatch),
        onSearch,
        isFetchingMore,
        fetchMoreError,
    };
}
