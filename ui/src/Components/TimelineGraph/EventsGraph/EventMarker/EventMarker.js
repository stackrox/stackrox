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
    name,
    type,
    uid,
    reason,
    timestamp,
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
            {type === eventTypes.POLICY_VIOLATION && (
                <PolicyViolationEvent name={name} type={type} timestamp={timestamp} width={size} />
            )}
            {type === eventTypes.PROCESS_ACTIVITY && (
                <ProcessActivityEvent
                    name={name}
                    type={type}
                    uid={uid}
                    timestamp={timestamp}
                    width={size}
                />
            )}
            {type === eventTypes.RESTART && (
                <RestartEvent name={name} type={type} timestamp={timestamp} width={size} />
            )}
            {type === eventTypes.TERMINATION && (
                <TerminationEvent
                    name={name}
                    type={type}
                    reason={reason}
                    timestamp={timestamp}
                    width={size}
                />
            )}
        </D3Anchor>
    );
};

export default EventMarker;
