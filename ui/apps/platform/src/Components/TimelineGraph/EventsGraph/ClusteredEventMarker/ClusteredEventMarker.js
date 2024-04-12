import React, { useRef } from 'react';
import PropTypes from 'prop-types';
import { scaleLinear } from 'd3-scale';
import uniq from 'lodash/uniq';

import { getWidth } from 'utils/d3Utils';
import { clusteredEventTypes } from 'constants/timelineTypes';
import mainViewSelector from 'Components/TimelineGraph/MainView/selectors';
import D3Anchor from 'Components/D3Anchor';
import ClusteredEventsTooltip from 'Components/TimelineGraph/EventsGraph/ClusteredEventsTooltip';
import ClusteredGenericEvent from './ClusteredGenericEvent';
import ClusteredPolicyViolationEvent from './ClusteredPolicyViolationEvent';
import ClusteredRestartEvent from './ClusteredRestartEvent';
import ClusteredProcessActivityEvent from './ClusteredProcessActivityEvent';
import ClusteredTerminationEvent from './ClusteredTerminationEvent';

/**
 * Determines the type of the clustered event based on the group of events
 * @param {Object[]} events
 * @returns {clusteredEventTypes}
 */
function getClusterEventType(events) {
    const types = uniq(
        events.map((event) =>
            // if the event is a process in baseline activity, we should use a new type specific
            // to just clustered events
            event.inBaseline ? clusteredEventTypes.PROCESS_IN_BASELINE_ACTIVITY : event.type
        )
    );
    // if all the events are the same type, use that type
    if (types.length === 1) {
        return types[0];
    }
    // if we have multiple types of events, use the generic type
    return clusteredEventTypes.GENERIC;
}

const ClusteredEventMarker = ({
    events,
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

    const clusterEventType = getClusterEventType(events);
    const numEvents = events.length;

    return (
        <D3Anchor
            dataTestId="timeline-clustered-event-marker"
            translateX={translateX}
            translateY={translateY}
            onUpdate={onUpdate}
        >
            <ClusteredEventsTooltip events={events} popoverRef={popoverRef}>
                <g>
                    {clusterEventType === clusteredEventTypes.GENERIC && (
                        <ClusteredGenericEvent size={size} numEvents={numEvents} ref={popoverRef} />
                    )}
                    {clusterEventType === clusteredEventTypes.POLICY_VIOLATION && (
                        <ClusteredPolicyViolationEvent
                            size={size}
                            numEvents={numEvents}
                            ref={popoverRef}
                        />
                    )}
                    {clusterEventType === clusteredEventTypes.PROCESS_ACTIVITY && (
                        <ClusteredProcessActivityEvent
                            size={size}
                            numEvents={numEvents}
                            ref={popoverRef}
                        />
                    )}
                    {clusterEventType === clusteredEventTypes.PROCESS_IN_BASELINE_ACTIVITY && (
                        <ClusteredProcessActivityEvent
                            size={size}
                            inBaseline
                            numEvents={numEvents}
                            ref={popoverRef}
                        />
                    )}
                    {clusterEventType === clusteredEventTypes.RESTART && (
                        <ClusteredRestartEvent size={size} numEvents={numEvents} ref={popoverRef} />
                    )}
                    {clusterEventType === clusteredEventTypes.TERMINATION && (
                        <ClusteredTerminationEvent
                            size={size}
                            numEvents={numEvents}
                            ref={popoverRef}
                        />
                    )}
                </g>
            </ClusteredEventsTooltip>
        </D3Anchor>
    );
};

// TODO: replace with `Event` TS type once this module is migrated to TypeScript
const eventPropTypes = {
    name: PropTypes.string.isRequired,
    args: PropTypes.string,
    type: PropTypes.string.isRequired,
    uid: PropTypes.number,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    inBaseline: PropTypes.bool,
};

ClusteredEventMarker.propTypes = {
    events: PropTypes.arrayOf(PropTypes.shape(eventPropTypes)).isRequired,
    differenceInMilliseconds: PropTypes.number.isRequired,
    translateX: PropTypes.number.isRequired,
    translateY: PropTypes.number.isRequired,
    minTimeRange: PropTypes.number.isRequired,
    maxTimeRange: PropTypes.number.isRequired,
    size: PropTypes.number.isRequired,
    margin: PropTypes.number,
};

ClusteredEventMarker.defaultProps = {
    margin: 0,
};

export default ClusteredEventMarker;
