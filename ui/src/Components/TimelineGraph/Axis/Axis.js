import React from 'react';
import { scaleLinear } from 'd3-scale';
import { axisTop, axisBottom } from 'd3-axis';

import selectors from 'Components/TimelineGraph/MainView/selectors';
import { getWidth } from 'utils/d3Utils';
import D3Anchor from 'Components/D3Anchor';

function timeDiffTickFormat(datum, i, values) {
    if (i !== 0 && i !== values.length - 1) {
        // since "datum" is the difference in hours, we want to get the whole number value for the hours value
        const hours = datum.toFixed(0);
        // (datum % 1) will get the fractional value from the difference in hours. We can use that to calculate the minute values
        const minutes = String(Math.round((datum % 1).toFixed(2) * 60)).padStart(2, '0');
        return `+${hours}:${minutes}h`;
    }
    return null;
}

function getAxisDirection(direction) {
    switch (direction) {
        case 'bottom':
            return axisBottom;
        default:
            return axisTop;
    }
}

export const AXIS_HEIGHT = 12;

const Axis = ({ translateX, translateY, minDomain, maxDomain, direction }) => {
    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const width = getWidth(selectors.svgSelector);
        const scale = scaleLinear()
            .domain([minDomain, maxDomain])
            .range([0, width]);
        const axis = getAxisDirection(direction)(scale)
            .tickFormat(timeDiffTickFormat)
            .tickSize(0);

        container.call(axis).call(g => g.select('.domain').style('stroke', 'var(--base-300)'));
    }
    return (
        <D3Anchor
            dataTestId="timeline-axis"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        />
    );
};

export default Axis;
