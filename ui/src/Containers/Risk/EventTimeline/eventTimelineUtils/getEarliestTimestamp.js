import { isBefore, parse } from 'date-fns';

/**
 * Returns the earliest timestamp from a collection of timestamps
 *
 * @param {string[]} timestamps - A collection of timestamps in the form '2020-04-24T16:00:00Z'
 * @returns {string}
 */
function getEarliestTimestamp(timestamps) {
    if (!timestamps || !timestamps.length) return null;
    const earliestTimestamp = timestamps.reduce((acc, curr) => {
        if (isBefore(parse(acc), parse(curr))) return acc;
        return curr;
    });
    return earliestTimestamp;
}

export default getEarliestTimestamp;
