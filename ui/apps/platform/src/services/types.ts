import { ApiSortOption } from 'types/search';

export type Pagination = {
    offset: number;
    limit: number;
    sortOption: ApiSortOption;
};

export type Empty = Record<string, never>;
