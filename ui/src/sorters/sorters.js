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
 * Sort Time
 * @param a
 * @param b
 * @returns {number}
 */
const sortTime = key => (a, b) => {
    const { aValue, bValue } = flattenObjectProperties(a, b, key);
    return new Date(bValue) - new Date(aValue);
};

/**
 * Sort Numbers by property name
 * @param key
 * @returns {number}
 */
const sortNumber = key => (a, b) => {
    const { aValue, bValue } = flattenObjectProperties(a, b, key);
    if (aValue === bValue) return 0;
    if (aValue === undefined) return -1;
    if (bValue === undefined) return 1;
    return aValue - bValue;
};

const sortDate = key => (a, b) => {
    const { aValue, bValue } = flattenObjectProperties(a, b, key);
    const aDate = aValue && new Date(aValue);
    const bDate = bValue && new Date(bValue);
    if (aDate === bDate) return 0;
    if (aDate === undefined) return -1;
    if (bValue === undefined) return 1;
    return aDate - bDate;
};
export { sortSeverity, sortTime, sortNumber, sortDate };
