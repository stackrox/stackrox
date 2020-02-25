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

const sortSeverity = (a, b) => {
    const map = {
        Low: 'LOW_SEVERITY',
        Medium: 'MEDIUM_SEVERITY',
        High: 'HIGH_SEVERITY',
        Critical: 'CRITICAL_SEVERITY'
    };
    const priorityArray = ['CRITICAL_SEVERITY', 'HIGH_SEVERITY', 'MEDIUM_SEVERITY', 'LOW_SEVERITY'];
    const firstSeverity = map[a] || a;
    const secondSeverity = map[b] || b;

    const firstPrio = priorityArray.indexOf(firstSeverity);
    const secPrio = priorityArray.indexOf(secondSeverity);
    return firstPrio - secPrio;
};

/**
 * Sort Status
 * @param a
 * @param b
 * @returns {number}
 */

const sortStatus = (a, b) => {
    const map = {
        Pass: 'PASS',
        NA: 'N/A',
        Fail: 'FAIL'
    };
    const priorityArray = ['FAIL', 'PASS', 'N/A'];
    const firstSeverity = map[a] || a;
    const secondSeverity = map[b] || b;

    const firstPrio = priorityArray.indexOf(firstSeverity);
    const secPrio = priorityArray.indexOf(secondSeverity);
    return firstPrio - secPrio;
};

/**
 * Sort Values (Numbers or Strings)
 * @returns {number}
 */
const sortValue = (a, b) => {
    if (a === undefined) return -1;
    if (b === undefined) return 1;
    const numA = Number(a);
    const numB = Number(b);
    if (!Number.isNaN(numA) && !Number.isNaN(numB)) {
        if (numA < numB) return -1;
        if (numA > numB) return 1;
        return 0;
    }
    if (a < b) return -1;
    if (a > b) return 1;
    return 0;
};

/**
 * Sort Version (Numbers or Strings)
 * @returns {number}
 */
const sortVersion = (a, b) => {
    if (a === undefined) return -1;
    if (b === undefined) return 1;
    const aSplit = a.split('.');
    const bSplit = b.split('.');

    const length = Math.min(aSplit.length, bSplit.length);
    for (let i = 0; i < length; i += 1) {
        if (parseInt(aSplit[i], 10) < parseInt(bSplit[i], 10)) {
            return -1;
        }
        if (parseInt(aSplit[i], 10) > parseInt(bSplit[i], 10)) {
            return 1;
        }
    }

    if (aSplit.length < bSplit.length) {
        return -1;
    }
    if (aSplit.length > bSplit.length) {
        return 1;
    }

    return 0;
};

/**
 * Sort Numbers by property name
 * @param key
 * @returns {number}
 */
const sortNumberByKey = key => (a, b) => {
    const { aValue, bValue } = flattenObjectProperties(a, b, key);
    return sortValue(aValue, bValue);
};

/**
 * Sort Lifecycle
 * @param a
 * @param b
 * @returns {string}
 */

const sortLifecycle = (a, b) => {
    const aValue = a[0];
    const bValue = b[0];
    return sortValue(aValue, bValue);
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

/**
 * Sort by array length
 * @returns {string}
 */
const sortValueByLength = (a, b) => {
    if (a === undefined) return -1;
    if (b === undefined) return 1;
    if (a.length > b.length) return 1;
    if (a.length < b.length) return -1;
    if (a.length === b.length) return 0;
    return a.length - b.length;
};

/**
 * Sort by ASCII, respects case
 */

/**
 * [sortAscii description]
 *
 * @param   {string}   a  first item to compare
 * @param   {string}  b  second item to compare
 *
 * @return  {number}     negative if a sorts before b, positive if b sorts before a, 0 if equal
 */
function sortAscii(a, b) {
    // NOTE: we are not doing the expected comparison
    //   a.localeCompare(b, 'en', { sensitivity: 'case' })
    // because that is case-insensitive, even with that given option
    //   which merely makes each letter sort A first, then a, but does not
    //   sort A-Z before a-z
    if (a < b) return -1;
    if (a > b) return 1;
    return 0;
}

export {
    sortSeverity,
    sortValue,
    sortStatus,
    sortVersion,
    sortNumberByKey,
    sortLifecycle,
    sortDate,
    sortValueByLength,
    sortAscii
};
