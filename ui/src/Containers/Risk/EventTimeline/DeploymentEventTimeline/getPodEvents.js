import { format, differenceInHours } from 'date-fns';

import { timelineStartTimeFormat } from 'constants/dateTimeFormat';
import { eventTypes, graphTypes } from 'constants/timelineTypes';

const filterByEventType = selectedEventType => event => {
    if (selectedEventType === eventTypes.ALL) return true;
    return event.type === selectedEventType;
};

const getPodEvents = (pods, selectedEventType) => {
    const podsWithEvents = pods.map(({ id, name, inactive, startTime, events, numContainers }) => ({
        type: graphTypes.POD,
        id,
        name,
        subText: inactive ? 'Inactive' : format(startTime, timelineStartTimeFormat),
        events: events
            .filter(filterByEventType(selectedEventType))
            .map(({ processId, timestamp, edges, type }) => ({
                id: processId,
                type,
                differenceInHours: differenceInHours(timestamp, startTime),
                edges
            })),
        hasChildren: numContainers > 0
    }));
    return podsWithEvents;
};

export default getPodEvents;
