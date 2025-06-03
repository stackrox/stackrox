import { useCallback } from 'react';
import useURLParameter, { HistoryAction } from './useURLParameter';

export type UseURLPaginationResult = {
    page: number;
    perPage: number;
    setPage: (page: number, historyAction?: HistoryAction) => void;
    setPerPage: (perPage: number, historyAction?: HistoryAction) => void;
};

function safeNumber(val: unknown, defaultVal: number) {
    const parsed = Number(val);

    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : defaultVal;
}

function useURLPagination(defaultPerPage: number, keyPrefix?: string): UseURLPaginationResult {
    const pageParam = keyPrefix ? `${keyPrefix}Page` : 'page';
    const perPageParam = keyPrefix ? `${keyPrefix}PerPage` : 'perPage';

    const [page, setPageString] = useURLParameter(pageParam, '1');
    const [perPage, setPerPageString] = useURLParameter(perPageParam, `${defaultPerPage}`);
    const setPage = useCallback(
        (num: number, historyAction?: HistoryAction) =>
            setPageString(num > 1 ? String(num) : undefined, historyAction),
        [setPageString]
    );
    const setPerPage = useCallback(
        (num: number, historyAction?: HistoryAction) => {
            // If the history action is 'replace', we replace both the perPage and page in-place.
            // If the history action is 'push', we push a new perPage and replace the page in
            // order to keep a single record on the history stack.
            setPerPageString(num !== defaultPerPage ? String(num) : undefined, historyAction);
            setPageString('1', 'replace');
        },
        [setPageString, setPerPageString, defaultPerPage]
    );
    return {
        page: safeNumber(page, 1),
        perPage: safeNumber(perPage, defaultPerPage),
        setPage,
        setPerPage,
    };
}

export default useURLPagination;
