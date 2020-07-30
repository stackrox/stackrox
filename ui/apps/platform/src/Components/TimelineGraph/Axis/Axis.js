import React from 'react';
import PropTypes from 'prop-types';
import { scaleLinear } from 'd3-scale';
import { axisTop, axisBottom } from 'd3-axis';

import mainViewSelector from 'Components/TimelineGraph/MainView/selectors';
import { getWidth } from 'utils/d3Utils';
import D3Anchor from 'Components/D3Anchor';
import getTimeDiffTickFormat from './getTimeDiffTickFormat';

const DIRECTIONS = ['bottom', 'top'];

function getAxisDirection(direction) {
    switch (direction) {
        case 'bottom':
            return axisBottom;
        default:
            return axisTop;
    }
}

export const AXIS_HEIGHT = 12;

const Axis = ({ translateX, translateY, minDomain, maxDomain, direction, margin }) => {
    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const width = getWidth(mainViewSelector);
        const minRange = margin;
        const maxRange = width - margin;
        const scale = scaleLinear().domain([minDomain, maxDomain]).range([minRange, maxRange]);
        const axis = getAxisDirection(direction)(scale)
            .ticks(8)
            .tickFormat(getTimeDiffTickFormat)
            .tickSize(0);

        container.call(axis).call((g) => g.select('.domain').style('stroke', 'var(--base-300)'));
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

Axis.propTypes = {
    margin: PropTypes.number,
    translateX: PropTypes.number,
    translateY: PropTypes.number,
    minDomain: PropTypes.number.isRequired,
    maxDomain: PropTypes.number.isRequired,
    direction: PropTypes.oneOf(DIRECTIONS),
};

Axis.defaultProps = {
    margin: 0,
    translateX: 0,
    translateY: 0,
    direction: 'top',
};

export default Axis;
