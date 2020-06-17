import React from 'react';
import PropTypes from 'prop-types';
import pluralize from 'pluralize';

import { eventTypes } from 'constants/timelineTypes';
import { eventPropTypes } from 'constants/propTypes/timelinePropTypes';
import Tooltip from 'Components/Tooltip';
import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import getTimeRangeOfEvents from './getTimeRangeOfEvents';
import ProcessActivityTooltipFields from '../EventTooltipFields/ProcessActivityTooltipFields';
import TerminationTooltipFields from '../EventTooltipFields/TerminationTooltipFields';
import DefaultTooltipFields from '../EventTooltipFields/DefaultTooltipFields';
import PolicyViolationEvent from '../EventMarker/PolicyViolationEvent';
import ProcessActivityEvent from '../EventMarker/ProcessActivityEvent';
import RestartEvent from '../EventMarker/RestartEvent';
import TerminationEvent from '../EventMarker/TerminationEvent';

const ClusteredEventsTooltip = ({ events, children }) => {
    const { timeRangeOfEvents, unit } = getTimeRangeOfEvents(events);
    const tooltipTitle = `${events.length} ${pluralize(
        'Event',
        events.length
    )} within ${timeRangeOfEvents} ${unit}`;
    const sections = events.map(
        ({ id, type, name, args, uid, parentName, parentUid, timestamp, reason, whitelisted }) => {
            let section = null;
            switch (type) {
                case eventTypes.PROCESS_ACTIVITY:
                    section = (
                        <ProcessActivityTooltipFields
                            name={name}
                            args={args}
                            uid={uid}
                            parentName={parentName}
                            parentUid={parentUid}
                            timestamp={timestamp}
                        />
                    );
                    break;
                case eventTypes.TERMINATION:
                    section = (
                        <TerminationTooltipFields
                            name={name}
                            reason={reason}
                            timestamp={timestamp}
                        />
                    );
                    break;
                default:
                    section = (
                        <DefaultTooltipFields name={name} type={type} timestamp={timestamp} />
                    );
            }
            return (
                <li key={id} className="flex border-b border-base-300 border-primary-400 py-2">
                    <div className="mt-1 mr-4">
                        {type === eventTypes.POLICY_VIOLATION && <PolicyViolationEvent size={15} />}
                        {type === eventTypes.PROCESS_ACTIVITY && (
                            <ProcessActivityEvent size={15} whitelisted={whitelisted} />
                        )}
                        {type === eventTypes.RESTART && <RestartEvent size={15} />}
                        {type === eventTypes.TERMINATION && <TerminationEvent size={15} />}
                    </div>
                    <div>{section}</div>
                </li>
            );
        }
    );
    const tooltipBody = <ul>{sections}</ul>;

    return (
        <Tooltip
            trigger="click"
            interactive
            appendTo={document.body}
            content={<DetailedTooltipOverlay title={tooltipTitle} body={tooltipBody} />}
        >
            {children}
        </Tooltip>
    );
};

ClusteredEventsTooltip.propTypes = {
    events: PropTypes.arrayOf(PropTypes.shape(eventPropTypes)),
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]).isRequired,
};

ClusteredEventsTooltip.defaultProps = {
    events: [],
};

export default ClusteredEventsTooltip;
