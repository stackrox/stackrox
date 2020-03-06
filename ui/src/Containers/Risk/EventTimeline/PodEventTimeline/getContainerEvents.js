import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { eventTypes, graphTypes } from 'constants/timelineTypes';

const filterByEventType = selectedEventType => event => {
    if (selectedEventType === eventTypes.ALL) return true;
    return event.type === selectedEventType;
};

export const getPod = ({ id, name, inactive, startTime }) => {
    return {
        type: graphTypes.POD,
        id,
        name,
        subText: inactive ? 'Inactive' : format(startTime, dateTimeFormat)
    };
};

export const getContainerEvents = (containers, selectedEventType) => {
    const containersWithEvents = containers.map(({ id, name, inactive, startTime, events }) => ({
        type: graphTypes.CONTAINER,
        id,
        name,
        subText: inactive ? 'Inactive' : format(startTime, dateTimeFormat),
        events: events
            .filter(filterByEventType(selectedEventType))
            .map(({ processId, timestamp, edges, type }) => ({
                id: processId,
                type,
                timestamp,
                edges
            })),
        hasChildren: false
    }));
    return containersWithEvents;
};
