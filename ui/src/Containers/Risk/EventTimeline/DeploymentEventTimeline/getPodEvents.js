import { format } from 'date-fns';

import { timelineStartTimeFormat } from 'constants/dateTimeFormat';
import { graphTypes } from 'constants/timelineTypes';
import getDifferenceInHours from '../eventTimelineUtils/getDifferenceInHours';
import filterByEventType from '../eventTimelineUtils/filterByEventType';

const getPodEvents = (pods, selectedEventType) => {
    const podsWithEvents = pods.map(
        ({ id, name, inactive, startTime, events, containerCount }) => ({
            type: graphTypes.POD,
            id,
            name,
            subText: inactive ? 'Inactive' : format(startTime, timelineStartTimeFormat),
            events: events
                .filter(filterByEventType(selectedEventType))
                .map(({ id: processId, uid, reason, timestamp, edges, type }) => ({
                    id: processId,
                    type,
                    uid,
                    reason,
                    differenceInHours: getDifferenceInHours(timestamp, startTime),
                    timestamp,
                    edges
                })),
            hasChildren: containerCount > 0
        })
    );
    return podsWithEvents;
};

export default getPodEvents;
