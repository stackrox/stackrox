import maxBy from 'lodash/maxBy';

/**
 * @typedef {Object} Event
 * @property {number} differenceInHours - The difference in hours between the event and it's entity's start time
 */

/**
 * @typedef {Object} TimelineData
 * @property {Event[]} events
 */

/**
 * Iterates through all events across Entities and returns the largest difference in hours
 * @param {TimelineData[]} data
 * @returns {number} the largest difference in hours
 */
const getLargestDifferenceInHours = data => {
    const largestDifferenceInHours = data.reduce((acc, curr) => {
        const eventWithMaxValue = maxBy(curr.events, event => event.differenceInHours);
        if (!eventWithMaxValue) return acc;
        return Math.max(acc, eventWithMaxValue.differenceInHours);
    }, 0);
    return largestDifferenceInHours;
};

export default getLargestDifferenceInHours;
