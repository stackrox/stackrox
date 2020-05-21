import React from 'react';
import PropTypes from 'prop-types';

import EventMarker from './EventMarker';

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
}) => {
    const eventMarkerSize = Math.max(0, height / 3);
    const eventMarkerOffsetY = Math.max(0, height / 2);
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
            {events.map(
                ({
                    id,
                    type,
                    name,
                    args,
                    uid,
                    parentName,
                    parentUid,
                    reason,
                    whitelisted,
                    differenceInMilliseconds,
                    timestamp,
                }) => (
                    <EventMarker
                        key={id}
                        name={name}
                        args={args}
                        uid={uid}
                        parentName={parentName}
                        parentUid={parentUid}
                        reason={reason}
                        type={type}
                        timestamp={timestamp}
                        whitelisted={whitelisted}
                        differenceInMilliseconds={differenceInMilliseconds}
                        translateX={translateX}
                        translateY={eventMarkerOffsetY}
                        size={eventMarkerSize}
                        minTimeRange={minTimeRange}
                        maxTimeRange={maxTimeRange}
                        margin={margin}
                    />
                )
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
};

EventsRow.defaultProps = {
    margin: 0,
    translateX: 0,
    translateY: 0,
    events: [],
    isOdd: false,
};

export default EventsRow;
