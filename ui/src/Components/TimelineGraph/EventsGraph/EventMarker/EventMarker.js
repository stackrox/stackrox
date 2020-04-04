import React from 'react';
import { scaleLinear } from 'd3-scale';

import { getWidth } from 'utils/d3Utils';
import { eventTypes } from 'constants/timelineTypes';
import selectors from 'Components/TimelineGraph/MainView/selectors';
import D3Anchor from 'Components/D3Anchor';
import PolicyViolationEvent from './PolicyViolationEvent';
import RestartEvent from './RestartEvent';
import ProcessActivityEvent from './ProcessActivityEvent';
import TerminationEvent from './TerminationEvent';

const EventMarker = ({
    type,
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
            {type === eventTypes.POLICY_VIOLATION && <PolicyViolationEvent width={size} />}
            {type === eventTypes.PROCESS_ACTIVITY && <ProcessActivityEvent width={size} />}
            {type === eventTypes.RESTART && <RestartEvent width={size} />}
            {type === eventTypes.TERMINATION && <TerminationEvent width={size} />}
        </D3Anchor>
    );
};

export default EventMarker;
