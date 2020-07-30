import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { graphTypes } from 'constants/timelineTypes';
import processEvents from '../eventTimelineUtils/processEvents';
import filterByEventType from '../eventTimelineUtils/filterByEventType';

export const getPod = ({ id, name, startTime }) => {
    const formattedTime = startTime ? format(startTime, dateTimeFormat) : 'N/A';
    return {
        type: graphTypes.POD,
        id,
        name,
        subText: formattedTime,
    };
};

export const getContainerEvents = (containers, selectedEventType) => {
    const containersWithEvents = containers.map(({ id, name, startTime, events }) => {
        const filteredEvents = events.filter(filterByEventType(selectedEventType));
        const formattedTime = startTime ? format(startTime, dateTimeFormat) : 'N/A';
        const processedEvents = processEvents(filteredEvents);
        return {
            type: graphTypes.CONTAINER,
            id,
            name,
            subText: formattedTime,
            events: processedEvents,
            hasChildren: false,
        };
    });
    return containersWithEvents;
};
