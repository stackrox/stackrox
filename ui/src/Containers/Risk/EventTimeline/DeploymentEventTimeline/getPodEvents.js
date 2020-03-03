import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { graphTypes } from 'constants/timelineTypes';

const getPodEvents = pods => {
    const podsWithEvents = pods.map(({ id, name, inactive, startTime, events, numContainers }) => ({
        type: graphTypes.POD,
        id,
        name,
        subText: inactive ? 'Inactive' : format(startTime, dateTimeFormat),
        events: events.map(({ processId, timestamp, edges, type }) => ({
            id: processId,
            type,
            timestamp,
            edges
        })),
        hasChildren: numContainers > 0
    }));
    return podsWithEvents;
};

export default getPodEvents;
