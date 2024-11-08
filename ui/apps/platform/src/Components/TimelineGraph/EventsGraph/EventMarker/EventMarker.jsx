import React, { useRef } from 'react';
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
    id,
    name,
    args,
    type,
    uid,
    parentName,
    parentUid,
    reason,
    timestamp,
    inBaseline,
    differenceInMilliseconds,
    translateX,
    translateY,
    minTimeRange,
    maxTimeRange,
    size,
    margin,
}) => {
    const popoverRef = useRef(null);

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
                id={id}
                name={name}
                args={args}
                type={type}
                uid={uid}
                parentName={parentName}
                parentUid={parentUid}
                timestamp={timestamp}
                reason={reason}
                inBaseline={inBaseline}
                popoverRef={popoverRef}
            >
                <g>
                    {type === eventTypes.POLICY_VIOLATION && (
                        <PolicyViolationEvent size={size} ref={popoverRef} />
                    )}
                    {type === eventTypes.PROCESS_ACTIVITY && (
                        <ProcessActivityEvent
                            size={size}
                            inBaseline={inBaseline}
                            ref={popoverRef}
                        />
                    )}
                    {type === eventTypes.RESTART && <RestartEvent size={size} ref={popoverRef} />}
                    {type === eventTypes.TERMINATION && (
                        <TerminationEvent size={size} ref={popoverRef} />
                    )}
                </g>
            </EventTooltip>
        </D3Anchor>
    );
};

EventMarker.propTypes = {
    id: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    args: PropTypes.string,
    type: PropTypes.string.isRequired,
    uid: PropTypes.number,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    inBaseline: PropTypes.bool,
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
    inBaseline: false,
    margin: 0,
};

export default EventMarker;
