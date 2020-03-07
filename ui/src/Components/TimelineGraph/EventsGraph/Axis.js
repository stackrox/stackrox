import React from 'react';
import { select } from 'd3-selection';
import { scaleLinear } from 'd3-scale';
import { axisTop } from 'd3-axis';

import selectors from 'Components/TimelineGraph/EventsGraph/selectors';
import D3Anchor from 'Components/D3Anchor';

function hourDiffTickFormat(datum, i, values) {
    if (i !== 0 && i !== values.length - 1) return `+${datum}h`;
    return null;
}

const Axis = ({ translateX, translateY }) => {
    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const width = parseInt(select(selectors.svgSelector).style('width'), 10);
        const scale = scaleLinear()
            .domain([0, 12]) // @TODO: This should be able to change once the brush logic comes in
            .range([0, width]);
        const axis = axisTop(scale)
            .tickFormat(hourDiffTickFormat)
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
