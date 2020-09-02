import { useMemo } from 'react';

import { getLastCategoryInSearchOptions } from 'Components/SearchInput';

/**
 * Returns a mapping of categories to values in the data associated with that category
 *
 * Example: { Traffic: ['bidirectional', 'ingress', 'egress'] }
 *
 * @param {!Object[]} data
 * @param {!String[]} categories
 * @param {!Function} getDataValueByCategory - function that returns the value in the data
 *                                             based on a category field
 * @returns {!Object}
 */
export function getAutoCompleteResultsByCategories(data, categories, getDataValueByCategory) {
    const autoCompleteResultsByCategories = data.reduce((acc, datum) => {
        categories.forEach((category) => {
            const value = getDataValueByCategory(datum, category);
            if (!acc[category]) {
                acc[category] = new Set();
            }
            // if the value is an array, we need to add each item to the set
            if (Array.isArray(value)) {
                value.forEach((item) => acc[category].add(item.toString()));
            } else {
                acc[category].add(value.toString());
            }
        });
        return acc;
    }, {});
    // convert the Set -> Array for each category
    Object.keys(autoCompleteResultsByCategories).forEach((category) => {
        autoCompleteResultsByCategories[category] = Array.from(
            autoCompleteResultsByCategories[category]
        );
    });
    return autoCompleteResultsByCategories;
}

/**
 * Returns a list of autocomplete results associated with the last category in the search options
 *
 * @param {!Object[]} data
 * @param {!Object[]} searchOptions
 * @param {!String[]} categories
 * @param {!Function} getDataValueByCategory - function that returns the value in the data
 *                                             based on a category field
 * @returns {!Object[]}
 */
function useAutoCompleteResults(data, searchOptions, categories, getDataValueByCategory) {
    const autoCompleteResultsByCategories = useMemo(
        () => getAutoCompleteResultsByCategories(data, categories, getDataValueByCategory),
        [categories, data, getDataValueByCategory]
    );
    const category = getLastCategoryInSearchOptions(searchOptions);
    const autoCompleteResults = autoCompleteResultsByCategories[category] || [];

    return autoCompleteResults;
}

export default useAutoCompleteResults;
