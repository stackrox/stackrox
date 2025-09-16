import React, { useContext } from 'react';
import PropTypes from 'prop-types';

import ClusteredEventsVisibilityContext from 'Components/TimelineGraph/ClusteredEventsVisibilityContext';
import GroupedEvents from './GroupedEvents';
import Events from './Events';

const EventsRow = ({
    entityName,
    events,
    isOdd,
    height,
    width,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange,
    margin,
    isZooming,
}) => {
    const showClusteredEvents = useContext(ClusteredEventsVisibilityContext);
    // filter events to only show those in the viewable window
    const eventsWithinView = events.filter(({ differenceInMilliseconds }) => {
        return differenceInMilliseconds >= minTimeRange && differenceInMilliseconds <= maxTimeRange;
    });

    return (
        <g
            data-testid="timeline-events-row"
            key={entityName}
            transform={`translate(${translateX}, ${translateY})`}
            fill="transparent"
        >
            <rect
                className="pointer-events-none"
                fill={isOdd ? 'var(--tertiary-200)' : 'var(--base-100)'}
                stroke="var(--base-300)"
                height={height}
                width={width}
            />
            {showClusteredEvents && (
                <GroupedEvents
                    events={eventsWithinView}
                    height={height}
                    width={width}
                    translateX={translateX}
                    minTimeRange={minTimeRange}
                    maxTimeRange={maxTimeRange}
                    margin={margin}
                    isZooming={isZooming}
                />
            )}
            {!showClusteredEvents && (
                <Events
                    events={eventsWithinView}
                    height={height}
                    translateX={translateX}
                    minTimeRange={minTimeRange}
                    maxTimeRange={maxTimeRange}
                    margin={margin}
                    isZooming={isZooming}
                />
            )}
        </g>
    );
};

EventsRow.propTypes = {
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    margin: PropTypes.number,
    height: PropTypes.number.isRequired,
    width: PropTypes.number.isRequired,
    translateX: PropTypes.number,
    translateY: PropTypes.number,
    entityName: PropTypes.string.isRequired,
    events: PropTypes.arrayOf(PropTypes.object),
    isOdd: PropTypes.bool,
    isZooming: PropTypes.bool,
};

EventsRow.defaultProps = {
    margin: 0,
    translateX: 0,
    translateY: 0,
    events: [],
    isOdd: false,
    isZooming: false,
};

export default EventsRow;
