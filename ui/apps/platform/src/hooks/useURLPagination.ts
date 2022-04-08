import { useCallback } from 'react';
import useURLParameter from './useURLParameter';

type UseURLPaginationResult = {
    page: number;
    perPage: number;
    setPage: (page: number) => void;
    setPerPage: (perPage: number) => void;
};

function useURLPagination(defaultPerPage: number): UseURLPaginationResult {
    const [page, setPageString] = useURLParameter<string | undefined>('page', '1');
    const [perPage, setPerPageString] = useURLParameter<string | undefined>(
        'perPage',
        `${defaultPerPage}`
    );
    const setPage = useCallback(
        (num: number) => setPageString(num > 1 ? String(num) : undefined),
        [setPageString]
    );
    const setPerPage = useCallback(
        (num: number) => setPerPageString(num !== defaultPerPage ? String(num) : undefined),
        [setPerPageString, defaultPerPage]
    );
    return {
        page: Number(page),
        perPage: Number(perPage),
        setPage,
        setPerPage,
    };
}

export default useURLPagination;
