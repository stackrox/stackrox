import React, { useState, useEffect, useRef } from 'react';

import { getWidth, getHeight } from 'utils/d3Utils';
import Axis, { AXIS_HEIGHT } from '../Axis';
import EventsGraph from '../EventsGraph';

const MainView = ({ data, minTimeRange, maxTimeRange, numRows }) => {
    const refAnchor = useRef(null);
    const [width, setWidth] = useState(0);
    const [height, setHeight] = useState(0);

    useEffect(() => {
        setWidth(getWidth(refAnchor.current));
        setHeight(getHeight(refAnchor.current));
    }, []);

    return (
        <svg data-testid="timeline-main-view" width="100%" height="100%" ref={refAnchor}>
            <Axis
                translateX={0}
                translateY={AXIS_HEIGHT}
                minDomain={minTimeRange}
                maxDomain={maxTimeRange}
            />
            <EventsGraph
                translateX={0}
                translateY={AXIS_HEIGHT}
                minTimeRange={minTimeRange}
                maxTimeRange={maxTimeRange}
                data={data}
                width={width}
                height={height}
                numRows={numRows}
            />
        </svg>
    );
};

export default MainView;
