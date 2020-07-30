import getDifferenceInMilliseconds from './getDifferenceInMilliseconds';
import getEarliestTimestamp from './getEarliestTimestamp';

/**
 * Processes the events data and returns a new array of events with modified data
 * @param {Object[]} events - The events data returned by the API call
 * @returns {Object[]} - The processed events data
 */
function processEvents(events) {
    const eventTimestamps = events.map((event) => event.timestamp);
    const earliestEventTimestamp = getEarliestTimestamp(eventTimestamps);
    return events.map((event) => ({
        ...event,
        differenceInMilliseconds: getDifferenceInMilliseconds(
            event.timestamp,
            earliestEventTimestamp
        ),
    }));
}

export default processEvents;
