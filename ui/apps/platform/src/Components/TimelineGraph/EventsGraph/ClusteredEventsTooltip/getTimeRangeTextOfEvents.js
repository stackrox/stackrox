import pluralize from 'pluralize';

import getTimeDiffWithUnit from 'Components/TimelineGraph/timelineGraphUtils/getTimeDiffWithUnit';

/**
 * Given an array of events, this function will give the time range between the earliest event
 * and the latest event in human readable text (ie. "1 hour and 2 minutes")
 * @param {object[]} events - The timeline events
 * @returns {string} - The human readable time range
 */
const getTimeRangeTextOfEvents = (events) => {
    const { min, max } = events.reduce(
        (acc, curr) => {
            let newMin = acc.min;
            let newMax = acc.max;
            if (curr.differenceInMilliseconds < acc.min) {
                newMin = curr.differenceInMilliseconds;
            }
            if (curr.differenceInMilliseconds > acc.max) {
                newMax = curr.differenceInMilliseconds;
            }
            return {
                min: newMin,
                max: newMax,
            };
        },
        { min: Infinity, max: -Infinity }
    );
    const timeRangeInMilliseconds = max - min;
    const timeDiffWithUnit = getTimeDiffWithUnit(timeRangeInMilliseconds);
    const timeRangeTextOfEvents = timeDiffWithUnit
        .filter((time) => time.timeDifference !== 0)
        .map((time) => {
            return `${time.timeDifference} ${pluralize(time.unit, time.timeDifference)}`;
        })
        .join(' and ');
    return timeRangeTextOfEvents || '0 ms';
};

export default getTimeRangeTextOfEvents;
