import React from 'react';
import { scaleLinear } from 'd3-scale';

import selectors from 'Components/TimelineGraph/MainView/selectors';
import { getWidth } from 'utils/d3Utils';
import D3Anchor from 'Components/D3Anchor';

const EventMarker = ({
    differenceInHours,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange,
    size
}) => {
    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const width = getWidth(selectors.svgSelector);
        const xScale = scaleLinear()
            .domain([minTimeRange, maxTimeRange])
            .range([0, width]);
        const x = xScale(differenceInHours).toFixed(0);

        container.attr(
            'transform',
            `translate(${Number(translateX) + Number(x) - size / 2}, ${Number(translateY) -
                size / 2})`
        );
    }
    return (
        <D3Anchor
            dataTestId="timeline-event-marker"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        >
            <rect fill="var(--primary-600)" width={size} height={size} />
        </D3Anchor>
    );
};

export default EventMarker;
