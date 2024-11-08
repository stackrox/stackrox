import React, { useState } from 'react';
import PropTypes from 'prop-types';

import EventsRow from './EventsRow';
import ZoomableOverlay from './ZoomableOverlay';

const MAX_ROW_HEIGHT = 48;
const MIN_ROW_HEIGHT = 0;

const EventsGraph = ({
    data,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange,
    absoluteMinTimeRange,
    absoluteMaxTimeRange,
    height,
    width,
    numRows,
    margin,
    isHeightAdjustable,
    onZoomChange,
}) => {
    const [isZooming, setZooming] = useState(false);

    const rowHeight = isHeightAdjustable
        ? Math.min(Math.max(MIN_ROW_HEIGHT, Math.floor(height / numRows) - 1), MAX_ROW_HEIGHT)
        : MAX_ROW_HEIGHT;

    function onZoomStart() {
        setZooming(true);
    }

    function onZoomEnd() {
        setZooming(false);
    }

    return (
        <g
            data-testid="timeline-events-graph"
            transform={`translate(${translateX}, ${translateY})`}
            height={rowHeight}
            width={width}
        >
            {onZoomChange && ( // we don't want to show this in the minimap
                <ZoomableOverlay
                    translateX={0}
                    translateY={0}
                    width={width}
                    height={height}
                    absoluteMinTimeRange={absoluteMinTimeRange}
                    absoluteMaxTimeRange={absoluteMaxTimeRange}
                    onZoomChange={onZoomChange}
                    onZoomStart={onZoomStart}
                    onZoomEnd={onZoomEnd}
                />
            )}
            {data.map((datum, index) => {
                const { id, name, events } = datum;
                const isOddRow = index % 2 !== 0;
                return (
                    <EventsRow
                        key={id}
                        entityName={name}
                        events={events}
                        isOdd={isOddRow}
                        height={rowHeight}
                        width={width}
                        translateX={0}
                        translateY={index * rowHeight}
                        minTimeRange={minTimeRange}
                        maxTimeRange={maxTimeRange}
                        margin={margin}
                        isZooming={isZooming}
                    />
                );
            })}
        </g>
    );
};

EventsGraph.propTypes = {
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    data: PropTypes.arrayOf(PropTypes.object).isRequired,
    numRows: PropTypes.number.isRequired,
    margin: PropTypes.number,
    height: PropTypes.number.isRequired,
    width: PropTypes.number.isRequired,
    translateX: PropTypes.number,
    translateY: PropTypes.number,
    isHeightAdjustable: PropTypes.bool,
    absoluteMinTimeRange: PropTypes.number,
    absoluteMaxTimeRange: PropTypes.number,
    onZoomChange: PropTypes.func,
};

EventsGraph.defaultProps = {
    margin: 0,
    translateX: 0,
    translateY: 0,
    isHeightAdjustable: false,
    onZoomChange: null,
    absoluteMinTimeRange: null,
    absoluteMaxTimeRange: null,
};

export default EventsGraph;
