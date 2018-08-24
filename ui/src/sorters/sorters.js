import flattenObject from 'utils/flattenObject';

const flattenObjectProperties = (a, b, key) => {
    const aValue = Object.assign({}, flattenObject(a))[key];
    const bValue = Object.assign({}, flattenObject(b))[key];
    return { aValue, bValue };
};

/**
 * Sort Severity
 * @param a
 * @param b
 * @returns {number}
 */

const sortSeverity = key => (a, b) => {
    const { aValue, bValue } = flattenObjectProperties(a, b, key);
    const map = {
        Low: 'LOW_SEVERITY',
        Medium: 'MEDIUM_SEVERITY',
        High: 'HIGH_SEVERITY',
        Critical: 'CRITICAL_SEVERITY'
    };
    const priorityArray = ['LOW_SEVERITY', 'MEDIUM_SEVERITY', 'HIGH_SEVERITY', 'CRITICAL_SEVERITY'];
    const firstSeverity = map[aValue] || aValue;
    const secondSeverity = map[bValue] || bValue;

    const firstPrio = priorityArray.indexOf(firstSeverity);
    const secPrio = priorityArray.indexOf(secondSeverity);
    return firstPrio - secPrio;
};

/**
 * Sort Numbers
 * @returns {number}
 */
const sortNumber = (a, b) => {
    if (a === b) return 0;
    if (a === undefined) return -1;
    if (b === undefined) return 1;
    return a - b;
};

/**
 * Placeholder function till migration of benchmarks table to react table
 * Sort Numbers by property name
 * @param key
 * @returns {number}
 */
const sortNumberByKey = key => (a, b) => {
    const { aValue, bValue } = flattenObjectProperties(a, b, key);
    return sortNumber(aValue, bValue);
};

/**
 * Sort Dates
 * @returns {date}
 */
const sortDate = (a, b) => {
    const aDate = a && new Date(a);
    const bDate = b && new Date(b);
    if (aDate === bDate) return 0;
    if (aDate === undefined) return -1;
    if (bDate === undefined) return 1;
    return aDate - bDate;
};
export { sortSeverity, sortNumber, sortNumberByKey, sortDate };
