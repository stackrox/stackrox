import React, { useState, useEffect } from 'react';
import { scaleLinear } from 'd3-scale';

import selectors from 'Components/TimelineGraph/Minimap/selectors';
import { getWidth, getHeight } from 'utils/d3Utils';
import EventsGraph from 'Components/TimelineGraph/EventsGraph';
import Axis, { AXIS_HEIGHT } from '../Axis';
import Brush from './Brush';

const MiniMap = ({
    minTimeRange,
    maxTimeRange,
    setMinTimeRange,
    setMaxTimeRange,
    data,
    numRows
}) => {
    const [width, setWidth] = useState(0);
    const [height, setHeight] = useState(0);

    useEffect(() => {
        setWidth(getWidth(selectors.svgSelector));
        setHeight(getHeight(selectors.svgSelector));
    }, []);

    function onSelectionChange(selection) {
        const scale = scaleLinear()
            .domain([minTimeRange, maxTimeRange])
            .range([0, width]);
        const newMinTimeRange = selection ? scale.invert(selection.start) : minTimeRange;
        const newMaxTimeRange = selection ? scale.invert(selection.end) : maxTimeRange;
        setMinTimeRange(newMinTimeRange);
        setMaxTimeRange(newMaxTimeRange);
    }

    const brushableViewHeight = Math.max(0, height - AXIS_HEIGHT);

    return (
        <svg data-testid="timeline-minimap" width="100%" height="100%">
            <EventsGraph
                translateX={0}
                translateY={0}
                minTimeRange={minTimeRange}
                maxTimeRange={maxTimeRange}
                data={data}
                width={width}
                height={brushableViewHeight}
                numRows={numRows}
            />
            <Brush
                translateX={0}
                translateY={0}
                width={width}
                height={brushableViewHeight}
                onSelectionChange={onSelectionChange}
            />
            <Axis
                translateX={0}
                translateY={brushableViewHeight}
                minDomain={minTimeRange}
                maxDomain={maxTimeRange}
                direction="bottom"
            />
        </svg>
    );
};

export default MiniMap;
