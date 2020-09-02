import intersection from 'lodash/intersection';

/**
 * Processes the search options and returns an array of categories with their associated
 * filter values
 *
 * Example:  [{ category: 'Fruit', values: ['Apple', 'Banana'] }]
 *
 * @param {!Object[]} searchOptions
 * @returns {!Object[]}
 */
export function getCategoryValuesPairs(searchOptions) {
    const { valuesByCategory } = searchOptions.reduce(
        (acc, curr) => {
            if (curr.type === 'categoryOption') {
                acc.currentCategory = curr.value.replace(':', '');
                acc.valuesByCategory[acc.currentCategory] = [];
            } else {
                acc.valuesByCategory[acc.currentCategory].push(curr.value);
            }
            return acc;
        },
        {
            currentCategory: null,
            valuesByCategory: {},
        }
    );
    return Object.keys(valuesByCategory).map((category) => {
        const values = valuesByCategory[category];
        return { category, values };
    });
}

/**
 * Filters an array based on the search options
 *
 * @param {!Object[]} data
 * @param {!Object[]} searchOptions
 * @param {!Function} getDataValueByCategory - function that returns the value in the data
 *                                             based on a category field
 * @returns {!Object[]}
 */
function useSearchFilteredData(data, searchOptions, getDataValueByCategory) {
    if (!searchOptions.length) {
        return data;
    }
    const categoryValuesPairs = getCategoryValuesPairs(searchOptions);
    const filteredResults = data.filter((datum) => {
        const hasMatchingValues = categoryValuesPairs.every(({ category, values }) => {
            if (!values || !values.length) {
                return true;
            }
            const value = getDataValueByCategory(datum, category);
            if (Array.isArray(value)) {
                return intersection(value, values).length !== 0;
            }
            if (values.indexOf(value) !== -1) {
                return true;
            }
            return false;
        });
        return hasMatchingValues;
    });
    return filteredResults;
}

export default useSearchFilteredData;
