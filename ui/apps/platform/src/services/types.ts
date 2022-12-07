import { ApiSortOption } from 'types/search';

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
