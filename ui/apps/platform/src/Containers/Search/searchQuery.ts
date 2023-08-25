import qs from 'qs';

import { SearchFilter } from 'types/search';

import { SearchNavCategory, searchNavMap } from './searchCategories';

export type SearchQueryObject = {
    searchFilter: SearchFilter;
    navCategory: SearchNavCategory;
};

type SearchQueryParse = {
    s: SearchFilter;
    category?: string;
};

export function parseQueryString(search: string, searchOptions: string[]): SearchQueryObject {
    const { category, s } = qs.parse(search, { ignoreQueryPrefix: true });

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

    return { searchFilter, navCategory };
}

export function stringifyQueryObject({ searchFilter, navCategory }: SearchQueryObject) {
    if (Object.keys(searchFilter).length === 0) {
        return '';
    }

    const queryObject: SearchQueryParse = { s: searchFilter };

    if (navCategory !== 'SEARCH_UNSET') {
        queryObject.category = searchNavMap[navCategory];
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
