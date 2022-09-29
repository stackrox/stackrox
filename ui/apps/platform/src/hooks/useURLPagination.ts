import { useCallback } from 'react';
import useURLParameter from './useURLParameter';

export type UseURLPaginationResult = {
    page: number;
    perPage: number;
    setPage: (page: number) => void;
    setPerPage: (perPage: number) => void;
};

function safeNumber(val: unknown, defaultVal: number) {
    const parsed = Number(val);

    return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : defaultVal;
}

function useURLPagination(defaultPerPage: number): UseURLPaginationResult {
    const [page, setPageString] = useURLParameter('page', '1');
    const [perPage, setPerPageString] = useURLParameter('perPage', `${defaultPerPage}`);
    const setPage = useCallback(
        (num: number) => setPageString(num > 1 ? String(num) : undefined),
        [setPageString]
    );
    const setPerPage = useCallback(
        (num: number) => setPerPageString(num !== defaultPerPage ? String(num) : undefined),
        [setPerPageString, defaultPerPage]
    );
    return {
        page: safeNumber(page, 1),
        perPage: safeNumber(perPage, defaultPerPage),
        setPage,
        setPerPage,
    };
}

export default useURLPagination;
