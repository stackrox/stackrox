import { graphTypes } from 'constants/timelineTypes';
import { getDateTime } from 'utils/dateUtils';
import processEvents from '../eventTimelineUtils/processEvents';
import filterByEventType from '../eventTimelineUtils/filterByEventType';

export const getPod = ({ id, name, startTime }) => {
    const formattedTime = startTime ? getDateTime(startTime) : 'N/A';
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
        const formattedTime = startTime ? getDateTime(startTime) : 'N/A';
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
