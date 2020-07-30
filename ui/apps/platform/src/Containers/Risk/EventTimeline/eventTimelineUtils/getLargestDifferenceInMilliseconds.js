import maxBy from 'lodash/maxBy';

/**
 * @typedef {Object} Event
 * @property {number} differenceInMillseconds - The difference in milliseconds between the event and it's entity's start time
 */

/**
 * @typedef {Object} TimelineData
 * @property {Event[]} events
 */

/**
 * Iterates through all events across Entities and returns the largest difference in milliseconds
 * @param {TimelineData[]} data
 * @returns {number} the largest difference in milliseconds
 */
const getLargestDifferenceInMilliseconds = (data) => {
    const largestDifferenceInMilliseconds = data.reduce((acc, curr) => {
        const eventWithMaxValue = maxBy(curr.events, (event) => event.differenceInMilliseconds);
        if (!eventWithMaxValue) return acc;
        return Math.max(acc, eventWithMaxValue.differenceInMilliseconds);
    }, 0);
    return largestDifferenceInMilliseconds;
};

export default getLargestDifferenceInMilliseconds;
