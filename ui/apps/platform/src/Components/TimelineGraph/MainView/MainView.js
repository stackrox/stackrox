import React, { useState, useEffect, useRef } from 'react';
import PropTypes from 'prop-types';

import { getWidth, getHeight } from 'utils/d3Utils';
import Axis, { AXIS_HEIGHT } from '../Axis';
import EventsGraph from '../EventsGraph';

const MainView = ({
    data,
    minTimeRange,
    maxTimeRange,
    absoluteMinTimeRange,
    absoluteMaxTimeRange,
    numRows,
    margin,
    onZoomChange,
}) => {
    const refAnchor = useRef(null);
    const [width, setWidth] = useState(0);
    const [height, setHeight] = useState(0);

    useEffect(() => {
        setWidth(getWidth(refAnchor.current));
        setHeight(getHeight(refAnchor.current));
    }, []);

    return (
        <svg data-testid="timeline-main-view" width="700px" height="500px" ref={refAnchor}>
            <Axis
                translateX={0}
                translateY={AXIS_HEIGHT}
                minDomain={minTimeRange}
                maxDomain={maxTimeRange}
                margin={margin}
            />
            <EventsGraph
                translateX={0}
                translateY={AXIS_HEIGHT}
                minTimeRange={minTimeRange}
                maxTimeRange={maxTimeRange}
                absoluteMinTimeRange={absoluteMinTimeRange}
                absoluteMaxTimeRange={absoluteMaxTimeRange}
                data={data}
                width={width}
                height={height}
                numRows={numRows}
                margin={margin}
                onZoomChange={onZoomChange}
            />
        </svg>
    );
};

MainView.propTypes = {
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    data: PropTypes.arrayOf(PropTypes.object).isRequired,
    numRows: PropTypes.number.isRequired,
    margin: PropTypes.number,
    absoluteMinTimeRange: PropTypes.number.isRequired,
    absoluteMaxTimeRange: PropTypes.number.isRequired,
    onZoomChange: PropTypes.func.isRequired,
};

MainView.defaultProps = {
    margin: 0,
};

export default MainView;
