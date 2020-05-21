import React from 'react';
import PropTypes from 'prop-types';
import { scaleLinear } from 'd3-scale';

import { getWidth } from 'utils/d3Utils';
import { eventTypes } from 'constants/timelineTypes';
import mainViewSelector from 'Components/TimelineGraph/MainView/selectors';
import D3Anchor from 'Components/D3Anchor';
import EventTooltip from 'Components/TimelineGraph/EventsGraph/EventTooltip';
import PolicyViolationEvent from './PolicyViolationEvent';
import RestartEvent from './RestartEvent';
import ProcessActivityEvent from './ProcessActivityEvent';
import TerminationEvent from './TerminationEvent';

const EventMarker = ({
    name,
    args,
    type,
    uid,
    parentName,
    parentUid,
    reason,
    timestamp,
    whitelisted,
    differenceInMilliseconds,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange,
    size,
    margin,
}) => {
    // the "container" argument is a reference to the container for the D3-related element
    function onUpdate(container) {
        const width = getWidth(mainViewSelector);
        const minRange = margin;
        const maxRange = width - margin;
        const xScale = scaleLinear()
            .domain([minTimeRange, maxTimeRange])
            .range([minRange, maxRange]);
        const x = xScale(differenceInMilliseconds).toFixed(0);

        container.attr(
            'transform',
            `translate(${Number(translateX) + Number(x) - size / 2}, ${
                Number(translateY) - size / 2
            })`
        );
    }

    return (
        <D3Anchor
            dataTestId="timeline-event-marker"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        >
            <EventTooltip
                name={name}
                args={args}
                type={type}
                uid={uid}
                parentName={parentName}
                parentUid={parentUid}
                timestamp={timestamp}
                reason={reason}
                whitelisted={whitelisted}
            >
                <g>
                    {type === eventTypes.POLICY_VIOLATION && <PolicyViolationEvent size={size} />}
                    {type === eventTypes.PROCESS_ACTIVITY && (
                        <ProcessActivityEvent size={size} whitelisted={whitelisted} />
                    )}
                    {type === eventTypes.RESTART && <RestartEvent size={size} />}
                    {type === eventTypes.TERMINATION && <TerminationEvent size={size} />}
                </g>
            </EventTooltip>
        </D3Anchor>
    );
};

EventMarker.propTypes = {
    name: PropTypes.string.isRequired,
    args: PropTypes.string,
    type: PropTypes.string.isRequired,
    uid: PropTypes.number,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    whitelisted: PropTypes.bool,
    differenceInMilliseconds: PropTypes.number.isRequired,
    translateX: PropTypes.number.isRequired,
    translateY: PropTypes.number.isRequired,
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    size: PropTypes.number.isRequired,
    margin: PropTypes.number,
};

EventMarker.defaultProps = {
    uid: null,
    parentName: null,
    parentUid: null,
    args: null,
    reason: null,
    whitelisted: false,
    margin: 0,
};

export default EventMarker;
