import type { ApiSortOption, ApiSortOptionSingle } from 'types/search';

/* The type for pagination data stored and generated client side */
export type ClientPagination = {
    page: number;
    perPage: number;
    sortOption?: ApiSortOption;
};

type PaginationBase = {
    offset: number;
    limit: number;
};

/* The type for pagination data passed to the server side APIs */
export type Pagination =
    | PaginationBase
    | (PaginationBase & { sortOption: ApiSortOptionSingle })
    | (PaginationBase & { sortOptions: ApiSortOptionSingle[] });

export type FilterQuery = {
    query: string;
    pagination: Pagination;
};

export type Empty = Record<string, never>;
