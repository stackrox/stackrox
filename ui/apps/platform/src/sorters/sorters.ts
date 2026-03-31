/**
 * Sort Severity
 * @param {string} a
 * @param {string} b
 * @returns {number}
 */

export const sortSeverity = (a: string, b: string) => {
    const map = {
        Low: 'LOW_SEVERITY',
        Medium: 'MEDIUM_SEVERITY',
        High: 'HIGH_SEVERITY',
        Critical: 'CRITICAL_SEVERITY',
    };
    const priorityArray = ['CRITICAL_SEVERITY', 'HIGH_SEVERITY', 'MEDIUM_SEVERITY', 'LOW_SEVERITY'];
    const firstSeverity = map[a] ?? a;
    const secondSeverity = map[b] ?? b;

    const firstPrio = priorityArray.indexOf(firstSeverity);
    const secPrio = priorityArray.indexOf(secondSeverity);
    return firstPrio - secPrio;
};

/**
 * Sort Status
 * @param {string} a
 * @param {string} b
 * @returns {number}
 */

export const sortStatus = (a: string, b: string) => {
    const map = {
        Pass: 'PASS',
        NA: 'N/A',
        Fail: 'FAIL',
    };
    const priorityArray = ['FAIL', 'PASS', 'N/A'];
    const firstSeverity = map[a] ?? a;
    const secondSeverity = map[b] ?? b;

    const firstPrio = priorityArray.indexOf(firstSeverity);
    const secPrio = priorityArray.indexOf(secondSeverity);
    return firstPrio - secPrio;
};

/**
 * Sort Values (Numbers or Strings)
 * @returns {number}
 */
export const sortValue = (a: number | string | undefined, b: number | string | undefined) => {
    if (a === undefined) {
        return -1;
    }
    if (b === undefined) {
        return 1;
    }
    const numA = Number(a);
    const numB = Number(b);
    if (!Number.isNaN(numA) && !Number.isNaN(numB)) {
        if (numA < numB) {
            return -1;
        }
        if (numA > numB) {
            return 1;
        }
        return 0;
    }
    if (a < b) {
        return -1;
    }
    if (a > b) {
        return 1;
    }
    return 0;
};

/**
 * Sort Version (Numbers or Strings)
 * @returns {number}
 */
export const sortVersion = (a: number | string | undefined, b: number | string | undefined) => {
    if (a === undefined) {
        return -1;
    }
    if (b === undefined) {
        return 1;
    }
    const aSplit = String(a).split('.');
    const bSplit = String(b).split('.');

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
 * Sort by array length
 * @returns {number}
 */
export const sortValueByLength = (a: string[] | undefined, b: string[] | undefined) => {
    if (a === undefined) {
        return -1;
    }
    if (b === undefined) {
        return 1;
    }
    if (a.length > b.length) {
        return 1;
    }
    if (a.length < b.length) {
        return -1;
    }
    if (a.length === b.length) {
        return 0;
    }
    return a.length - b.length;
};

/**
 * [sortAsciiCaseInsensitive description]
 *
 * @param   {string}   a  first item to compare
 * @param   {string}  b  second item to compare
 *
 * @return  {number}     negative if a sorts before b, positive if b sorts before a, 0 if equal
 */
export function sortAsciiCaseInsensitive(a: string, b: string) {
    return a.localeCompare(b, 'en', { sensitivity: 'base' });
}
