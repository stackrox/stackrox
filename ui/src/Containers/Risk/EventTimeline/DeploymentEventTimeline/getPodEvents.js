import { format, differenceInMilliseconds, parse } from 'date-fns';

import { timelineStartTimeFormat } from 'constants/dateTimeFormat';
import { eventTypes, graphTypes } from 'constants/timelineTypes';

const filterByEventType = selectedEventType => event => {
    if (selectedEventType === eventTypes.ALL) return true;
    return event.type === selectedEventType;
};

// assumes both date's are in the ISO 8601 format (2011-08-12T20:17:46.384Z)
function getDifferenceInHours(dateLeft, dateRight) {
    return differenceInMilliseconds(parse(dateLeft), parse(dateRight)) / 3600000;
}

const getPodEvents = (pods, selectedEventType) => {
    const podsWithEvents = pods.map(({ id, name, inactive, startTime, events, numContainers }) => ({
        type: graphTypes.POD,
        id,
        name,
        subText: inactive ? 'Inactive' : format(startTime, timelineStartTimeFormat),
        events: events
            .filter(filterByEventType(selectedEventType))
            .map(({ id: processId, timestamp, edges, type }) => ({
                id: processId,
                type,
                differenceInHours: getDifferenceInHours(timestamp, startTime),
                edges
            })),
        hasChildren: numContainers > 0
    }));
    return podsWithEvents;
};

export default getPodEvents;
