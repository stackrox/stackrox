import React from 'react';

import EventsRow from './EventsRow';

const MAX_ROW_HEIGHT = 48;
const MIN_ROW_HEIGHT = 0;

const EventsGraph = ({
    data,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange,
    height,
    width,
    numRows
}) => {
    const rowHeight = Math.min(
        Math.max(MIN_ROW_HEIGHT, Math.floor(height / numRows) - 1),
        MAX_ROW_HEIGHT
    );
    return (
        <g
            data-testid="timeline-events-graph"
            transform={`translate(${translateX}, ${translateY})`}
        >
            {data.map((datum, index) => {
                const { id, name, events } = datum;
                const isOddRow = index % 2 !== 0;
                return (
                    <EventsRow
                        key={id}
                        name={name}
                        events={events}
                        isOdd={isOddRow}
                        height={rowHeight}
                        width={width}
                        translateX={0}
                        translateY={index * rowHeight}
                        minTimeRange={minTimeRange}
                        maxTimeRange={maxTimeRange}
                    />
                );
            })}
        </g>
    );
};

export default EventsGraph;
