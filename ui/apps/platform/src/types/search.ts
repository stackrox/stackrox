import { AggregateFunc } from './table';

/*
 * Examples of search filter object properties parsed from search query string:
 * 'Lifecycle Stage': 'BUILD' from 's[Lifecycle Stage]=BUILD
 * 'Lifecycle Stage': ['BUILD', 'DEPLOY'] from 's[Lifecycle Stage]=BUILD&s[Lifecycle State]=DEPLOY'
 */
export type SearchFilter = Partial<Record<string, string | string[]>>;

/*
 * For array values of SearchInput props: searchModifiers and searchOptions.
 *
 * A categoryOption entry whose value and label properties end with a colon
 * corresponds to an option string without a colon.
 * For example 'Lifecycle Stage:' corresponds to 'Lifecycle Stage'
 */
export type SearchEntry = {
    type?: 'categoryOption';
    value: string; // an option ends with a colon
    label: string; // an option ends with a colon
};

export type ApiSortOption = {
    field: string;
    aggregateBy?: {
        aggregateFunc: AggregateFunc;
        distinct?: boolean;
    };
    reversed: boolean;
};

export type GraphQLSortOption = {
    id: string;
    desc: boolean;
};
