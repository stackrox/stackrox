import React from 'react';

import EventMarker from './EventMarker';

const EventsRow = ({
    name,
    events,
    isOdd,
    height,
    width,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange
}) => {
    const eventMarkerSize = Math.max(0, height / 3);
    const eventMarkerOffsetY = Math.max(0, height / 2);
    return (
        <g
            data-testid="timeline-events-row"
            key={name}
            transform={`translate(${translateX}, ${translateY})`}
        >
            <rect
                fill={isOdd ? 'var(--tertiary-200)' : 'var(--base-100)'}
                stroke="var(--base-300)"
                height={height}
                width={width}
            />
            {events.map(({ id, type, uid, reason, differenceInMilliseconds, timestamp }) => (
                <EventMarker
                    key={id}
                    name={name}
                    uid={uid}
                    reason={reason}
                    type={type}
                    timestamp={timestamp}
                    differenceInMilliseconds={differenceInMilliseconds}
                    translateX={translateX}
                    translateY={eventMarkerOffsetY}
                    size={eventMarkerSize}
                    minTimeRange={minTimeRange}
                    maxTimeRange={maxTimeRange}
                />
            ))}
        </g>
    );
};

export default EventsRow;
