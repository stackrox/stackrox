import React from 'react';
import { scaleLinear } from 'd3-scale';
import { axisTop, axisBottom } from 'd3-axis';

import selectors from 'Components/TimelineGraph/MainView/selectors';
import { getWidth } from 'utils/d3Utils';
import D3Anchor from 'Components/D3Anchor';
import getTimeDiffTickFormat from './getTimeDiffTickFormat';

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
            .ticks(8)
            .tickFormat(getTimeDiffTickFormat)
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
