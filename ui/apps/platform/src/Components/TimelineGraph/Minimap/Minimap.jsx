import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';

import miniMapSelector from 'Components/TimelineGraph/Minimap/selectors';
import { getWidth, getHeight } from 'utils/d3Utils';
import EventsGraph from 'Components/TimelineGraph/EventsGraph';
import Axis, { AXIS_HEIGHT } from '../Axis';
import BrushableOverlay from './BrushableOverlay';

const MiniMap = ({
    minTimeRange,
    maxTimeRange,
    minBrushTimeRange,
    maxBrushTimeRange,
    data,
    numRows,
    margin,
    onBrushSelectionChange,
}) => {
    const [width, setWidth] = useState(0);
    const [height, setHeight] = useState(0);

    useEffect(() => {
        setWidth(getWidth(miniMapSelector));
        setHeight(getHeight(miniMapSelector));
    }, []);

    const brushableViewHeight = Math.max(0, height - AXIS_HEIGHT);

    return (
        <svg data-testid="timeline-minimap" width="700px" height="150px">
            <EventsGraph
                translateX={0}
                translateY={0}
                minTimeRange={minTimeRange}
                maxTimeRange={maxTimeRange}
                data={data}
                width={width}
                height={brushableViewHeight}
                numRows={numRows}
                margin={margin}
                isHeightAdjustable
            />
            <BrushableOverlay
                translateX={0}
                translateY={0}
                width={width}
                height={brushableViewHeight}
                minTimeRange={minBrushTimeRange}
                maxTimeRange={maxBrushTimeRange}
                absoluteMinTimeRange={minTimeRange}
                absoluteMaxTimeRange={maxTimeRange}
                onBrushSelectionChange={onBrushSelectionChange}
                margin={margin}
            />
            <Axis
                translateX={0}
                translateY={brushableViewHeight}
                minDomain={minTimeRange}
                maxDomain={maxTimeRange}
                direction="bottom"
                margin={margin}
            />
        </svg>
    );
};

MiniMap.propTypes = {
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    onBrushSelectionChange: PropTypes.func.isRequired,
    data: PropTypes.arrayOf(PropTypes.object).isRequired,
    numRows: PropTypes.number.isRequired,
    margin: PropTypes.number,
    minBrushTimeRange: PropTypes.number.isRequired,
    maxBrushTimeRange: PropTypes.number.isRequired,
};

MiniMap.defaultProps = {
    margin: 0,
};

export default MiniMap;
