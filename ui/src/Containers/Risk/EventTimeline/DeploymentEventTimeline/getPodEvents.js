import { format } from 'date-fns';

import { timelineStartTimeFormat } from 'constants/dateTimeFormat';
import { graphTypes } from 'constants/timelineTypes';
import getDifferenceInHours from '../eventTimelineUtils/getDifferenceInHours';
import filterByEventType from '../eventTimelineUtils/filterByEventType';

const getPodEvents = (pods, selectedEventType) => {
    const podsWithEvents = pods.map(({ id, name, inactive, startTime, events, liveInstances }) => ({
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
                timestamp,
                edges
            })),
        hasChildren: liveInstances.length > 0
    }));
    return podsWithEvents;
};

export default getPodEvents;
