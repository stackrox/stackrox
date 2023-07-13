import { useState, useCallback, useEffect, useMemo } from 'react';
import debounce from 'lodash/debounce';

function filterNextPage<Item, ItemKey>(
    results: Item[],
    itemKeys: Set<ItemKey>,
    dedupKeyFn: ((item: Item) => ItemKey) | undefined
) {
    // If a deduplication function has been provided, use it to ensure only unique items
    // are added to the data results. Otherwise, return the results directly without the need
    // to update the key Set.
    if (!dedupKeyFn) {
        return { nextPageItems: results, nextKeys: itemKeys };
    }
    const nextPageItems: Item[] = [];
    const nextKeys = new Set(itemKeys);
    results.forEach((item) => {
        const itemKey = dedupKeyFn(item);
        if (!nextKeys.has(itemKey)) {
            nextKeys.add(itemKey);
            nextPageItems.push(item);
        }
    });
    return { nextPageItems, nextKeys };
}

export type UsePaginatedQueryReturn<Item> = {
    /** The returned data. Each item in the top level array is a page of data `Item`s */
    data: Item[][];
    /** The current page */
    page: number;
    /** If the last fetch resulted in an error the `Error` value will appear here */
    lastFetchError: Error | null;
    /** If a page fetch is in flight */
    isFetchingNextPage: boolean;
    /** If a page fetch is in flight due to the caller refreshing the entire paginated results array */
    isRefreshingResults: boolean;
    /** If no more results are present on the server */
    isEndOfResults: boolean;
    /** Imperatively request the next page, which will declaratively be available in `data` */
    fetchNextPage: (immediate?: boolean) => void;
    /** Clears the entire `data` array and fetches the first page from the beginning */
    resetPages: () => void;
    /** Clears the entire `data` array and does not fetch a new page */
    clearPages: () => void;
};

/**
 * This hook is used for infinite loading query patterns via pagination. It's primary use case is a UI list
 * that we want to infinitely load via something like a "View more" button, that shows all resulting data in
 * a single UI. i.e. It does not use traditional "paged" UI components.
 *
 * In addition to fetching the data, this hook also can handle deduplicating the response in cases where the source
 * data changes faster than it is loaded in the UI. This is needed to avoid React key conflicts when rendering the
 * data, as well as for ensuring the results are unique.
 *
 * Conceptually it is similar to tanstack-query's https://tanstack.com/query/v4/docs/guides/infinite-queries or
 * SWR's https://swr.vercel.app/docs/pagination#useswrinfinite so that we can replace this logic if we move
 * to library based querying in the future.
 */
export function usePaginatedQuery<Item, ItemKey>(
    /** The function used to fetch each page of data */
    queryFn: (page: number) => Promise<Item[]>,
    /** The number of items to retrieve per page */
    pageSize: number,
    initialOptions: {
        /** How much debounce delay should occur before the fetch request is sent */
        debounceRate?: number;
        /**
         * The function used to determine the unique key of each item to avoid duplication. If
         * this parameter is omitted the hook will allow duplicate data in the results.
         */
        dedupKeyFn?: (item: Item) => ItemKey;
        /** Callback function to fire if a request results in an error */
        onError?: (err: Error) => void;
        /**
         *  Whether or not the initial fetch call for the first page should be automatic or require
         * a manual call to `fetchNextPage`
         */
        manualFetch?: boolean;
    } = {}
): UsePaginatedQueryReturn<Item> {
    // These initialOptions are stored in state instead of being destructured directly since
    // the `fetchPageHandler` function below relies on the reference of `dedupKeyFn`. By storing these
    // in state, it removes the burden of wrapping in a useCallback or useMemo from the caller.
    const [{ debounceRate, dedupKeyFn, onError, manualFetch = false }] = useState(initialOptions);
    const [isRefreshingResults, setIsRefreshingResults] = useState(!manualFetch);
    const [isFetchingNextPage, setIsFetchingNextPage] = useState(!manualFetch);
    const [isEndOfResults, setIsEndOfResults] = useState(false);
    const [lastFetchError, setLastFetchError] = useState<Error | null>(null);

    const [itemKeys, setItemKeys] = useState<Set<ItemKey>>(new Set());
    const [itemPages, setItemPages] = useState<Item[][]>([]);

    // Pages are zero-indexed, so the length of the current cached data array is equal
    // to the current page
    const page = itemPages.length;

    const fetchPageHandler = useCallback(
        (fetchFn: typeof queryFn, nextPage: number, nextItemKeys: Set<ItemKey>) => {
            return fetchFn(nextPage)
                .then((res) => {
                    setLastFetchError(null);
                    if (res.length < pageSize) {
                        setIsEndOfResults(true);
                    }
                    const { nextPageItems, nextKeys } = filterNextPage(
                        res,
                        nextItemKeys,
                        dedupKeyFn
                    );
                    setItemPages((prevData) => {
                        const nextData = [...prevData];
                        // We set directly via array index here instead of appending to prevent bugs when
                        // additional fetch requests are initiated before the first completes.
                        nextData[nextPage] = nextPageItems;
                        return nextData;
                    });
                    setItemKeys(nextKeys);
                })
                .catch((err) => {
                    setLastFetchError(err);
                    if (onError) {
                        onError(err);
                    }
                })
                .finally(() => {
                    setIsFetchingNextPage(false);
                    setIsRefreshingResults(false);
                });
        },
        [dedupKeyFn, onError, pageSize]
    );

    // Retain a reference to the debounced fetch function and the raw fetch function for cases
    // we know that we want to fetch the data _now_.
    // (e.g. We would want to debounce for text input `keypress` events, but a single
    // "View more" button that is immediately disabled once clicked can fetch without delay.)
    const pageFetcher = useMemo(() => {
        const fetcherFn = (query, nextPage, keys) => {
            setIsFetchingNextPage(true);
            return fetchPageHandler(query, nextPage, keys);
        };
        return {
            debounced: debounce(fetcherFn, debounceRate ?? 0),
            immediate: fetcherFn,
        };
    }, [debounceRate, fetchPageHandler]);

    const fetchNextPage = useCallback(
        (immediate = false) => {
            if (immediate) {
                return pageFetcher.immediate(queryFn, page, itemKeys);
            }
            return pageFetcher.debounced(queryFn, page, itemKeys);
        },
        [pageFetcher, queryFn, page, itemKeys]
    );

    const clearPages = useCallback(() => {
        setItemPages([]);
        setItemKeys(new Set());
        setIsEndOfResults(false);
        setLastFetchError(null);
    }, []);

    const resetPages = useCallback(() => {
        clearPages();
        setIsRefreshingResults(true);
        setIsFetchingNextPage(true);
        // Error handling and state setting already complete, so ignore this error
        // eslint-disable-next-line @typescript-eslint/no-floating-promises
        pageFetcher.debounced(queryFn, 0, new Set());
    }, [clearPages, pageFetcher, queryFn]);

    useEffect(() => {
        if (!manualFetch) {
            resetPages();
        }
    }, [manualFetch, resetPages]);

    return {
        data: itemPages,
        page,
        lastFetchError,
        isFetchingNextPage,
        isRefreshingResults,
        isEndOfResults,
        fetchNextPage,
        resetPages,
        clearPages,
    };
}
