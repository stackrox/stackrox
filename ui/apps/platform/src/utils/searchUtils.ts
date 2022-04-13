import {
    SearchEntry,
    GlobalSearchOption,
    ApiSortOption,
    GraphQLSortOption,
    SearchFilter,
} from 'types/search';

/**
 *  Adds a search modifier to the searchOptions
 *
 *  @param searchOptions an array of search options
 *  @param modifier a modifier term (ie. 'Cluster:')
 *  @returns the modified search options
 */
export function addSearchModifier(
    searchOptions: GlobalSearchOption[],
    modifier: string
): GlobalSearchOption[] {
    const chip = { value: modifier, label: modifier, type: 'categoryOption' };
    return [...searchOptions, chip];
}

/**
 *  Adds a search keyword to the searchOptions
 *
 *  @param searchOptions an array of search options
 *  @param keyword a keyword term (ie. 'remote')
 *  @returns the modified search options
 */
export function addSearchKeyword(
    searchOptions: GlobalSearchOption[],
    keyword: string
): GlobalSearchOption[] {
    const chip = { value: keyword, label: keyword, className: 'Select-create-option-placeholder' };
    return [...searchOptions, chip];
}

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

/*
 * Return request query string for search filter. Omit filter criterion:
 * If option does not have value.
 */
export function getRequestQueryStringForSearchFilter(searchFilter: SearchFilter): string {
    return Object.entries(searchFilter)
        .filter(([, value]) => value.length !== 0)
        .map(([key, value]) => `${key}:${Array.isArray(value) ? value.join(',') : value}`)
        .join('+');
}
