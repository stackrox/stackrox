import React from 'react';

import Axis from './Axis';
import Rows from './Rows';

const EventsGraph = ({ data }) => {
    return (
        <svg data-testid="timeline-events-graph" width="100%" height="100%">
            <Axis translateX={0} translateY={12} />
            <Rows translateX={0} translateY={12} data={data} />
        </svg>
    );
};

export default EventsGraph;
