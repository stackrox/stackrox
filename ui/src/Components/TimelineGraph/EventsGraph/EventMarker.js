import React from 'react';
import { select } from 'd3-selection';
import { scaleLinear } from 'd3-scale';

import selectors from 'Components/TimelineGraph/EventsGraph/selectors';
import D3Anchor from 'Components/D3Anchor';

const markerWidth = 15;
const markerHeight = 15;

const EventMarker = ({ differenceInHours, translateX, translateY }) => {
    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const width = parseInt(select(selectors.svgSelector).style('width'), 10);
        const xScale = scaleLinear()
            .domain([0, 12])
            .range([0, width]);
        const x = xScale(differenceInHours).toFixed(0);

        container.attr(
            'transform',
            `translate(${Number(translateX) + Number(x) - markerWidth / 2}, ${translateY -
                markerHeight / 2})`
        );
    }
    return (
        <D3Anchor
            dataTestId="timeline-event-marker"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        >
            <rect fill="var(--primary-600)" width={markerWidth} height={markerHeight} />
        </D3Anchor>
    );
};

export default EventMarker;
