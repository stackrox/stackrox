import qs from 'qs';

import {
    SearchEntry,
    ApiSortOption,
    GraphQLSortOption,
    SearchFilter,
    ApiSortOptionSingle,
} from 'types/search';
import { Pagination } from 'services/types';
import { ValueOf } from './type.utils';
import { safeGeneratePath } from './urlUtils';

/**
 *  Checks if the modifier exists in the searchOptions
 *
 *  @param {!Object[]} searchOptions an array of search options
 *  @param {!string} modifier
 *  @returns {boolean}
 */
export function hasSearchModifier(searchOptions, modifier) {
    return !!searchOptions.find(
        (option) => option.type === 'categoryOption' && option.value === modifier
    );
}

export function getViewStateFromSearch(
    search: Record<string, string | boolean>,
    key: string
): boolean {
    return !!(
        key &&
        search &&
        Object.keys(search).find((searchItem) => searchItem === key) && // key has to be present in current search criteria
        search[key] !== false &&
        search[key] !== 'false'
    ); // and the value of the search for that key cannot be false or the string "false", see https://stack-rox.atlassian.net/browse/ROX-4278
}

export function filterAllowedSearch(
    allowed: string[] = [],
    currentSearch: SearchFilter = {}
): Record<string, string> {
    const filtered = Object.keys(currentSearch)
        .filter((key) => allowed.includes(key))
        .reduce((newSearch, key) => {
            return {
                ...newSearch,
                [key]: currentSearch[key],
            };
        }, {});

    return filtered;
}

export function convertToRestSearch(workflowSearch: Record<string, string>): SearchEntry[] {
    const emptyArray: SearchEntry[] = [];
    if (!workflowSearch) {
        return emptyArray;
    }

    const restSearch = Object.keys(workflowSearch).reduce((acc, key) => {
        const keyWithColon = `${key}:`;
        const value = workflowSearch[key];

        const searchOption: SearchEntry = {
            label: keyWithColon,
            value: keyWithColon,
            type: 'categoryOption',
        };
        const searchValue = { label: value, value: value || '' };

        return searchValue.value ? acc.concat(searchOption, searchValue) : acc;
    }, emptyArray);

    return restSearch;
}

export function convertSortToGraphQLFormat({
    field,
    reversed,
}: ApiSortOptionSingle): GraphQLSortOption {
    return {
        id: field,
        desc: reversed,
    };
}

export function convertSortToRestFormat(
    graphqlSort: GraphQLSortOption[]
): Partial<ApiSortOptionSingle> {
    return {
        field: graphqlSort[0]?.id,
        reversed: graphqlSort[0]?.desc,
    };
}

/**
 * Function to convert the legacy SearchEntry array format to the
 * SearchFilter format.
 */
export function searchOptionsToSearchFilter(searchOptions: SearchEntry[]): SearchFilter {
    const searchFilter = {};
    let currentOption = '';
    searchOptions.forEach(({ value, type }) => {
        if (type === 'categoryOption') {
            // categoryOption represents the key of a search filter
            const option = value.replace(':', '');
            searchFilter[option] = '';
            currentOption = option;
        } else if (searchFilter[currentOption].length === 0) {
            // If this is the first search value for this category, store it as a string
            searchFilter[currentOption] = value;
        } else if (!Array.isArray(searchFilter[currentOption])) {
            // If this is not the first search value for this category, store it in a new array
            searchFilter[currentOption] = [searchFilter[currentOption], value];
        } else {
            // If we already have an array, simply add the next value
            searchFilter[currentOption].push(value);
        }
    });
    return searchFilter;
}

/**
 * Determines whether or not a SearchFilter contains a valid value for
 * all keys. A valid value is either a non-empty string or non-empty array.
 */
export function isCompleteSearchFilter(searchFilter: SearchFilter) {
    return Object.values(searchFilter).every(
        (o) => Boolean(o) && (!Array.isArray(o) || o.length > 0)
    );
}

/**
 * Type Guard to determine if a 2-tuple SearchFilter entry contains a non-empty value
 */
function isNonEmptySearchEntry<Key>(
    entry: [Key, string | string[] | undefined]
): entry is [Key, string | string[]] {
    return typeof entry[1] !== 'undefined' && entry[1].length !== 0;
}

/*
 * Return request query string for search filter. Omit filter criterion:
 * If option does not have value.
 */
export function getRequestQueryStringForSearchFilter(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .filter(isNonEmptySearchEntry)
        .map(([key, value]) => `${key}:${Array.isArray(value) ? value.join(',') : value}`)
        .join('+');
}

export function getUrlQueryStringForSearchFilter(
    searchFilter: SearchFilter,
    searchPrefix = 's'
): string {
    return qs.stringify(
        { [searchPrefix]: searchFilter },
        {
            arrayFormat: 'repeat',
            encodeValuesOnly: true,
        }
    );
}

/**
 * Helper function to determine if any search has been applied.
 *
 * @param searchFilter The `SearchFilter` value to check.
 *
 * @returns boolean, true if there are any search params
 */
export function getHasSearchApplied(searchFilter: SearchFilter): boolean {
    return Boolean(Object.keys(searchFilter).length);
}

/**
 * Helper function to flatten the value from a `SearchFilter` into a single Array.
 * Array state values stored in the URL are coerced into a singular `string` if they contain
 * one item, or are `undefined` if the key is not part of the `SearchFilter`.
 *
 * @param value The `SearchFilter` value to flatten.
 * @param fallback Fallback value to use if `value` is undefined. Typically this will be an empty array.
 *
 * @returns A one-dimensional array of strings, or the `fallback` value if input is undefined
 */
export function flattenFilterValue<UndefinedFallback>(
    value: string | string[] | undefined,
    fallback: UndefinedFallback
): string[] | UndefinedFallback {
    if (typeof value === 'undefined') {
        return fallback;
    }
    if (Array.isArray(value)) {
        return value;
    }
    return [value];
}

/**
 * Function to convert the standard list API pagination and query parameters into a
 * URL query string.
 *
 * @param options.searchFilter The `SearchFilter` to apply to the list query
 * @param options.sortOption The field to sort results by and whether to sort ascending or descending
 * @param options.page The page offset to return, pages are 1-indexed
 * @param options.perPage The number of items per page
 */
export function getListQueryParams({
    searchFilter,
    sortOption,
    page,
    perPage,
}: {
    searchFilter: SearchFilter;
    sortOption: ApiSortOption;
    page: number;
    perPage: number;
}): string {
    const query = getRequestQueryStringForSearchFilter(searchFilter);
    return qs.stringify(
        {
            query,
            pagination: getPaginationParams({ page, perPage, sortOption }),
        },
        { allowDots: true }
    );
}

/**
 * Calculates the API pagination limit and offset parameters given the
 * current page and number of items per page
 */
export function getPaginationParams({
    page,
    perPage,
    sortOption,
}: {
    page: number;
    perPage: number;
    sortOption?: ApiSortOption;
}): Pagination {
    const safePage = Math.max(1, page); // Prevent negative page numbers, page numbers are 1-indexed
    const safePerPage = Math.max(0, perPage); // Prevent negative perPage values
    const paginationBase = {
        offset: (safePage - 1) * safePerPage,
        limit: safePerPage,
    };

    if (typeof sortOption === 'undefined') {
        return paginationBase;
    }

    // When using multiple sort options, the API expects an array of sort options and the
    // plural form of `sortOption` is used.
    if (Array.isArray(sortOption)) {
        return { ...paginationBase, sortOptions: sortOption };
    }

    return { ...paginationBase, sortOption };
}

/**
 * Coerces a search filter value obtained from the URL into an array of strings.
 *
 * Array values will be returned unchanged.
 * String values will return an array of length one.
 * undefined values will return an empty array.
 *
 * @param searchValue The value of a single key from a `SearchFilter`
 * @returns An array of strings
 */
export function searchValueAsArray(searchValue: ValueOf<SearchFilter>): string[] {
    if (!searchValue) {
        return [];
    }
    if (Array.isArray(searchValue)) {
        return searchValue;
    }
    return [searchValue];
}

/**
 * Adds the StackRox bespoke flag for regex match, plus start-of-line and end-of-line character
 *
 * Non-string values will be returned unchanged.
 * String values will return as "r/^<original>$".
 *
 * @param {string} item
 * @returns {string}
 */
export function convertToExactMatch(item): unknown {
    if (typeof item !== 'string') {
        return item;
    }
    return `r/^${item}$`;
}

/**
 * Adds acs regex flag to values in the searchFilter object
 *
 * All values are prefixed by default
 * If keysToTransform is provided, only those keys will be modified
 *
 * @param {Object} searchFilter Original searchFilter object
 * @param {Array<string>} [keysToTransform] Optional – The keys in the searchFilter object to transform
 * @returns {Object} New SearchFilter object where values (determined by keysToTransform) are prefixed with 'r/'
 */
export function addRegexPrefixToFilters(
    searchFilter: SearchFilter,
    keysToTransform: string[] | null = null
) {
    const modifiedFilter: SearchFilter = {};

    Object.keys(searchFilter).forEach((key) => {
        const value = searchFilter[key];
        const shouldTransform = !keysToTransform || keysToTransform.includes(key);

        if (shouldTransform) {
            if (Array.isArray(value)) {
                modifiedFilter[key] = value.map((item) => `r/${item}`);
            } else {
                modifiedFilter[key] = `r/${value}`;
            }
        } else {
            modifiedFilter[key] = value;
        }
    });

    return modifiedFilter;
}

// Uses the generatePath function from react-router in addition to adding the query params
// TODO: Fallback needed?
export const generatePathWithQuery = (
    pathTemplate: string,
    pathParams: Partial<Record<string, unknown>>,
    options: {
        customParams?: string | URLSearchParams | string[][] | Record<string, string>;
        searchFilter?: SearchFilter;
    } = {}
): string => {
    const { customParams = {}, searchFilter = {} } = options;
    const path = safeGeneratePath(pathTemplate, pathParams, pathTemplate);
    const customParamsString = new URLSearchParams(customParams).toString();
    const searchFilterString = getUrlQueryStringForSearchFilter(searchFilter);
    const queryParams = [customParamsString, searchFilterString].filter(Boolean).join('&');

    return queryParams ? `${path}?${queryParams}` : path;
};
