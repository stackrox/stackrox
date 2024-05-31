import { ApiSortOption } from 'types/search';

/* The type for pagination data stored and generated client side */
export type ClientPagination = {
    page: number;
    perPage: number;
    sortOption?: ApiSortOption;
};

/* The type for pagination data passed to the server side APIs */
export type Pagination = {
    offset: number;
    limit: number;
    sortOption?: ApiSortOption;
};

export type FilterQuery = {
    query: string;
    pagination: Pagination;
};

export type Empty = Record<string, never>;
