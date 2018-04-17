import flattenObject from 'utils/flattenObject';

/**
 * Sort Severity
 * @param a
 * @param b
 * @returns {number}
 */

export function sortSeverity(a, b) {
    const map = {
        Low: 'LOW_SEVERITY',
        Medium: 'MEDIUM_SEVERITY',
        High: 'HIGH_SEVERITY',
        Critical: 'CRITICAL_SEVERITY'
    };
    const priorityArray = ['LOW_SEVERITY', 'MEDIUM_SEVERITY', 'HIGH_SEVERITY', 'CRITICAL_SEVERITY'];
    const firstSeverity = map[a.severity] || a.severity;
    const secondSeverity = map[b.severity] || b.severity;

    const firstPrio = priorityArray.indexOf(firstSeverity);
    const secPrio = priorityArray.indexOf(secondSeverity);
    return firstPrio - secPrio;
}

/**
 * Sort Time
 * @param a
 * @param b
 * @returns {number}
 */

export function sortTime(a, b) {
    return new Date(b.time) - new Date(a.time);
}

/**
 * Sort Numbers by property name
 * @param key
 * @returns {number}
 */
const sortNumber = key => (a, b) => {
    const aValue = Object.assign({}, flattenObject(a))[key];
    const bValue = Object.assign({}, flattenObject(b))[key];
    if (aValue === bValue) return 0;
    if (aValue === undefined) return -1;
    if (bValue === undefined) return 1;
    return aValue - bValue;
};
export { sortNumber };

const sortDate = key => (a, b) => {
    const aValue = Object.assign({}, flattenObject(a))[key];
    const bValue = Object.assign({}, flattenObject(b))[key];
    const aDate = aValue && new Date(aValue);
    const bDate = bValue && new Date(bValue);
    if (aDate === bDate) return 0;
    if (aDate === undefined) return -1;
    if (bValue === undefined) return 1;
    return aDate - bDate;
};
export { sortDate };
