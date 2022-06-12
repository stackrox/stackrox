import qs from 'qs';
import { SearchEntry, ApiSortOption, GraphQLSortOption, SearchFilter } from 'types/search';

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
    currentSearch: Record<string, string> = {}
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

export function convertSortToGraphQLFormat({ field, reversed }: ApiSortOption): GraphQLSortOption {
    return {
        id: field,
        desc: reversed,
    };
}

export function convertSortToRestFormat(graphqlSort: GraphQLSortOption[]): Partial<ApiSortOption> {
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
