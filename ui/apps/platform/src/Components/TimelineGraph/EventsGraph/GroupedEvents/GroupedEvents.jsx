import React from 'react';
import PropTypes from 'prop-types';

import getGroupedEvents from './getGroupedEvents';
import Events from '../Events';
import ClusteredEventMarker from '../ClusteredEventMarker';

const GroupedEvents = ({
    events,
    height,
    width,
    translateX,
    minTimeRange,
    maxTimeRange,
    margin,
    isZooming,
}) => {
    if (isZooming) {
        return null;
    }

    const clusteredEventMarkerSize = Math.max(0, height / 2);
    const clusteredEventMarkerOffsetY = Math.max(0, height / 2);

    const groupedEvents = getGroupedEvents({
        events,
        minDomain: minTimeRange,
        maxDomain: maxTimeRange,
        minRange: margin,
        maxRange: Math.max(margin, width - margin),
        partitionSize: clusteredEventMarkerSize,
    });

    return (
        <>
            {groupedEvents.map((group) => {
                const {
                    differenceInMilliseconds: groupedDifferenceInMilliseconds,
                    events: eventsFromGroup,
                } = group;
                // if there is more than one event in the group, we will render a clustered event
                if (eventsFromGroup.length > 1) {
                    return (
                        <ClusteredEventMarker
                            key={groupedDifferenceInMilliseconds}
                            events={eventsFromGroup}
                            differenceInMilliseconds={groupedDifferenceInMilliseconds}
                            translateX={translateX}
                            translateY={clusteredEventMarkerOffsetY}
                            size={clusteredEventMarkerSize}
                            minTimeRange={minTimeRange}
                            maxTimeRange={maxTimeRange}
                            margin={margin}
                        />
                    );
                }
                return (
                    <Events
                        key={groupedDifferenceInMilliseconds}
                        events={eventsFromGroup}
                        height={height}
                        translateX={translateX}
                        minTimeRange={minTimeRange}
                        maxTimeRange={maxTimeRange}
                        margin={margin}
                        isZooming={isZooming}
                    />
                );
            })}
        </>
    );
};

GroupedEvents.propTypes = {
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    margin: PropTypes.number,
    height: PropTypes.number.isRequired,
    width: PropTypes.number.isRequired,
    translateX: PropTypes.number,
    events: PropTypes.arrayOf(PropTypes.object),
    isZooming: PropTypes.bool,
};

GroupedEvents.defaultProps = {
    margin: 0,
    translateX: 0,
    events: [],
    isZooming: false,
};

export default GroupedEvents;
