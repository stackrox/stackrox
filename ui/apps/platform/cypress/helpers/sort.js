/*
 * Assert that each pair of adjacent items are in sorted order.
 * items might be DOM elements from cy.get(…) call.
 * callbackForPairOfSortedItems(itemA, itemB) is returned from higher-order function below.
 */
export function assertSortedItems(items, callbackForPairOfSortedItems) {
    for (let indexB = 1; indexB < items.length; indexB += 1) {
        const itemA = items[indexB - 1];
        const itemB = items[indexB];
        const result = callbackForPairOfSortedItems(itemA, itemB);

        // No news is good news. One incorrect result is enough bad news.
        if (result !== '') {
            expect(result).to.equal('');
            break;
        }
    }
}

/*
 * Create callback function for assertSortedItems function above.
 * notValidDescription looks like a function name
 * - Its prefix is not.
 * - Its root is a description in PascalCase of valid and sorted values from items.
 * getValueFromItem encapsulates access like item?.innerText property.
 * isValidValue encapsulates expected type of value like string.
 * isPairOfSortedValues encapsulates comparison like ascending or descending.
 *
 * Return empty string for a pair of valid and sorted values.
 * Return non-empty string which looks like a function call for a pair of invalid or unsorted values.
 */
export function createCallbackForPairOfSortedItems(
    notValidDescription,
    getValueFromItem,
    isValidValue,
    isPairOfSortedValues
) {
    return function callbackForPairOfItems(itemA, itemB) {
        const valueA = getValueFromItem(itemA);
        const valueB = getValueFromItem(itemB);

        if (isValidValue(valueA) && isValidValue(valueB) && isPairOfSortedValues(valueA, valueB)) {
            return ''; // return empty string for correctly sorted pair
        }

        return `${notValidDescription}(${valueA}, ${valueB})`;
    };
}

export function getNumberValueFromElement(element) {
    return Number(element?.innerText);
}

export function isValidNumberValue(value) {
    return typeof value === 'number' && !Number.isNaN(value);
}

// Compare either number or string values.
export function isPairOfAscendingValues(valueA, valueB) {
    return valueA <= valueB;
}

// Compare either number or string values.
export function isPairOfDescendingValues(valueA, valueB) {
    return valueA >= valueB;
}

export const callbackForPairOfAscendingNumberValuesFromElements =
    createCallbackForPairOfSortedItems(
        'notAscendingNumberValuesFromElements',
        getNumberValueFromElement,
        isValidNumberValue,
        isPairOfAscendingValues
    );

export const callbackForPairOfDescendingNumberValuesFromElements =
    createCallbackForPairOfSortedItems(
        'notDescendingNumberValuesFromElements',
        getNumberValueFromElement,
        isValidNumberValue,
        isPairOfDescendingValues
    );

export function getStringValueFromElement(element) {
    return element?.innerText;
}

const severityValues = ['Low', 'Medium', 'High', 'Critical'];

export function isValidSeverityValue(value) {
    return typeof value === 'string' && severityValues.includes(value);
}

export function isPairOfAscendingSeverityValues(valueA, valueB) {
    const indexA = severityValues.indexOf(valueA);
    const indexB = severityValues.indexOf(valueB);
    return indexA !== -1 && indexA <= indexB;
}

export function isPairOfDescendingSeverityValues(valueA, valueB) {
    const indexA = severityValues.indexOf(valueA);
    const indexB = severityValues.indexOf(valueB);
    return indexA >= indexB && indexB !== -1;
}

export const callbackForPairOfAscendingSeverityValuesFromElements =
    createCallbackForPairOfSortedItems(
        'notAscendingSeverityValuesFromElements',
        getStringValueFromElement,
        isValidSeverityValue,
        isPairOfAscendingSeverityValues
    );

export const callbackForPairOfDescendingSeverityValuesFromElements =
    createCallbackForPairOfSortedItems(
        'notDescendingSeverityValuesFromElements',
        getStringValueFromElement,
        isValidSeverityValue,
        isPairOfDescendingSeverityValues
    );
