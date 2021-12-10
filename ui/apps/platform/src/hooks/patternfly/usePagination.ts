import { useState } from 'react';

type Pagination = {
    page: number;
    perPage: number;
};

export type UsePaginationResult = {
    page: number;
    perPage: number;
    onSetPage: (event, page: number) => void;
    onPerPageSelect: (event, perPage: number) => void;
};

function usePagination(): UsePaginationResult {
    const [pagination, setPagination] = useState<Pagination>({
        page: 1,
        perPage: 20,
    });

    function onSetPage(_, page) {
        setPagination((prevResult) => {
            return {
                ...prevResult,
                page,
            };
        });
    }

    function onPerPageSelect(_, perPage) {
        setPagination((prevResult) => {
            return {
                ...prevResult,
                perPage,
            };
        });
    }

    return { ...pagination, onSetPage, onPerPageSelect };
}

export default usePagination;
