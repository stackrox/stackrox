import useURLSearchState from './useURLSearchState';

export type PageOption = {
    page: string;
    perPage: string;
};

export type UsePaginationResult = {
    page: number;
    perPage: number;
    onSetPage: (event, page: number) => void;
    onPerPageSelect: (event, perPage: number) => void;
};

function usePagination(): UsePaginationResult {
    const [pageOption, setPageOption] = useURLSearchState<PageOption>('pageOption');

    // get the page option values from the URL, if available
    // otherwise, use the default sort option values
    const page = pageOption?.page ? Number(pageOption?.page) : 1;
    const perPage = pageOption?.perPage ? Number(pageOption?.perPage) : 20;

    function onSetPage(_, newPage) {
        const newPageOption: PageOption = {
            page: newPage,
            perPage: String(perPage),
        };
        setPageOption(newPageOption);
    }

    function onPerPageSelect(_, newPerPage) {
        const newPageOption: PageOption = {
            page: String(page),
            perPage: newPerPage,
        };
        setPageOption(newPageOption);
    }

    return { page, perPage, onSetPage, onPerPageSelect };
}

export default usePagination;
