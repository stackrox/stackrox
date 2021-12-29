import { useState } from 'react';

export type UsePaginationResult = {
    page: number;
    perPage: number;
    onSetPage: (event, page: number) => void;
    onPerPageSelect: (event, perPage: number) => void;
};

function usePagination(): UsePaginationResult {
    const [page, setPage] = useState(1);
    const [perPage, setPerPage] = useState(20);

    function onSetPage(_, newPage) {
        setPage(newPage);
    }

    function onPerPageSelect(_, newPerPage) {
        setPerPage(newPerPage);
    }

    return { page, perPage, onSetPage, onPerPageSelect };
}

export default usePagination;
