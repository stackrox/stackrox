import pluralize from 'pluralize';

import { getDateTime } from 'utils/dateUtils';
import { graphTypes } from 'constants/timelineTypes';
import processEvents from '../eventTimelineUtils/processEvents';
import filterByEventType from '../eventTimelineUtils/filterByEventType';

const getPodEvents = (pods, selectedEventType) => {
    const podsWithEvents = pods.map(({ id, name, inactive, startTime, events, containerCount }) => {
        const filteredEvents = events.filter(filterByEventType(selectedEventType));
        const formattedTime = inactive ? 'Inactive' : `Started  ${getDateTime(startTime)}`;
        const processedEvents = processEvents(filteredEvents);
        const hasContainers = containerCount > 0;
        return {
            type: graphTypes.POD,
            id,
            name,
            subText: formattedTime,
            events: processedEvents,
            hasChildren: hasContainers,
            drillDownButtonTooltip: `View ${containerCount} ${pluralize(
                'Container',
                containerCount
            )}`,
        };
    });
    return podsWithEvents;
};

export default getPodEvents;
