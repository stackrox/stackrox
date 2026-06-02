import { useState } from 'react';

/**
 * Manages page/perPage state and computes the offset for API calls.
 */
export function usePagination(defaultPerPage = 20) {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(defaultPerPage);

    const offset = (page - 1) * perPage;

    function onSetPage(_: unknown, newPage: number) {
        setPage(newPage);
    }

    function onPerPageSelect(_: unknown, newPerPage: number) {
        setPerPage(newPerPage);
        setPage(1);
    }

    return { page, perPage, offset, onSetPage, onPerPageSelect };
}
