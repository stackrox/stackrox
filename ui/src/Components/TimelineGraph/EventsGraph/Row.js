import React from 'react';

import EventMarker from './EventMarker';

const Row = ({ name, events, isOdd, height, width, translateX, translateY }) => {
    return (
        <g data-testid="timeline-row" key={name}>
            <rect
                fill={isOdd ? 'var(--tertiary-200)' : 'var(--base-100)'}
                stroke="var(--base-300)"
                height={height}
                width={width}
                transform={`translate(${translateX}, ${translateY})`}
            />
            {events.map(({ id, differenceInHours }) => (
                <EventMarker
                    key={id}
                    name={name}
                    differenceInHours={differenceInHours}
                    translateX={translateX}
                    translateY={translateY - height / 2}
                />
            ))}
        </g>
    );
};

export default Row;
