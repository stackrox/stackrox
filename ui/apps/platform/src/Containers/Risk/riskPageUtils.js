export function filterAllowedSearch(allowed = [], currentSearch = {}) {
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

export function convertToRestSearch(workflowSearch) {
    if (!workflowSearch) return [];

    const restSearch = Object.keys(workflowSearch).reduce((acc, key) => {
        const keyWithColon = `${key}:`;
        const value = workflowSearch[key];

        const searchOption = { label: keyWithColon, value: keyWithColon, type: 'categoryOption' };
        const searchValue = { label: value, value: value || '' };

        return searchValue.value ? acc.concat(searchOption, searchValue) : acc;
    }, []);

    return restSearch;
}

export function convertSortToGraphQLFormat({ field, reversed }) {
    return {
        id: field,
        desc: reversed,
    };
}

export function convertSortToRestFormat(graphqlSort) {
    return {
        field: graphqlSort[0]?.id,
        reversed: graphqlSort[0]?.desc,
    };
}
