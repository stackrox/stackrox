import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { graphTypes } from 'constants/timelineTypes';
import getDifferenceInMilliseconds from '../eventTimelineUtils/getDifferenceInMilliseconds';
import filterByEventType from '../eventTimelineUtils/filterByEventType';

export const getPod = ({ id, name, startTime }) => {
    return {
        type: graphTypes.POD,
        id,
        name,
        subText: startTime ? format(startTime, dateTimeFormat) : 'N/A'
    };
};

export const getContainerEvents = (containers, selectedEventType) => {
    const containersWithEvents = containers.map(({ id, name, startTime, events }) => ({
        type: graphTypes.CONTAINER,
        id,
        name,
        subText: startTime ? format(startTime, dateTimeFormat) : 'N/A',
        events: events
            .filter(filterByEventType(selectedEventType))
            .map(({ id: processId, uid, reason, timestamp, edges, type }) => ({
                id: processId,
                type,
                uid,
                reason,
                differenceInMilliseconds: getDifferenceInMilliseconds(timestamp, startTime),
                timestamp,
                edges
            })),
        hasChildren: false
    }));
    return containersWithEvents;
};
