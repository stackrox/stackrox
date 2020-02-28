import { format } from 'date-fns';

import dateTimeFormat from 'constants/dateTimeFormat';
import { graphTypes } from 'constants/timelineTypes';

const getPodEvents = pods => {
    const podsWithEvents = pods.map(({ id, name, inactive, startTime, events }) => ({
        type: graphTypes.POD,
        id,
        name,
        subText: inactive || format(startTime, dateTimeFormat),
        events: events.map(({ processId, timestamp, edges, type }) => ({
            id: processId,
            type,
            timestamp,
            edges
        }))
    }));
    return podsWithEvents;
};

export default getPodEvents;
