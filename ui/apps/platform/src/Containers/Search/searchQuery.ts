import qs from 'qs';

import type { SearchFilter } from 'types/search';

import { searchNavMap } from './searchCategories';
import type { SearchNavCategory } from './searchCategories';

export type SearchQueryObject = {
    searchFilter: SearchFilter;
    navCategory: SearchNavCategory;
    // Page size selected in the results table. Preserved across filter and category
    // changes so that those changes reset to page 1 without resetting the page size.
    perPage?: string;
};

type SearchQueryParse = {
    s: SearchFilter;
    category?: string;
    perPage?: string;
};

export function parseQueryString(search: string, searchOptions: string[]): SearchQueryObject {
    const { category, perPage, s } = qs.parse(search, { ignoreQueryPrefix: true });

    const searchFilter = {};
    if (typeof s === 'object') {
        Object.entries(s).forEach(([key, value]) => {
            if (searchOptions.includes(key)) {
                if (typeof value === 'string' || Array.isArray(value)) {
                    if (value.length !== 0) {
                        searchFilter[key] = value;
                    }
                }
            }
        });
    }

    let navCategory: SearchNavCategory = 'SEARCH_UNSET';
    if (Object.keys(searchFilter).length !== 0 && typeof category === 'string') {
        const navCategoryFound = Object.keys(searchNavMap).find(
            (navCategoryFinding) => category === searchNavMap[navCategoryFinding]
        );
        if (navCategoryFound) {
            navCategory = navCategoryFound as SearchNavCategory;
        }
    }

    const result: SearchQueryObject = { searchFilter, navCategory };
    if (typeof perPage === 'string') {
        result.perPage = perPage;
    }

    return result;
}

export function stringifyQueryObject({ searchFilter, navCategory, perPage }: SearchQueryObject) {
    if (Object.keys(searchFilter).length === 0) {
        return '';
    }

    const queryObject: SearchQueryParse = { s: searchFilter };

    if (navCategory !== 'SEARCH_UNSET') {
        queryObject.category = searchNavMap[navCategory];
    }

    // Carry the page size forward. The page parameter is intentionally omitted so that
    // changing the filter or category resets to page 1 while keeping the page size.
    if (typeof perPage === 'string') {
        queryObject.perPage = perPage;
    }

    return qs.stringify(queryObject, {
        addQueryPrefix: true,
        arrayFormat: 'repeat',
        encodeValuesOnly: true,
    });
}

export function parseSearchFilter(stringifiedSearchFilter): SearchFilter {
    const { s } = qs.parse(stringifiedSearchFilter, { ignoreQueryPrefix: true });

    if (typeof s === 'object') {
        return s as SearchFilter;
    }

    return {};
}

export function stringifySearchFilter(searchFilter: SearchFilter) {
    const queryObject: SearchQueryParse = { s: searchFilter };

    return qs.stringify(queryObject, {
        addQueryPrefix: false,
        arrayFormat: 'repeat',
        encodeValuesOnly: true,
    });
}
