import React, { ReactElement } from 'react';
import pluralize from 'pluralize';
import { Popover, Text, TextVariants } from '@patternfly/react-core';

import DetailedTooltipContent from 'Components/DetailedTooltipContent';
import { eventTypes } from 'constants/timelineTypes';
import { Event } from '../eventTypes';
import getTimeRangeTextOfEvents from './getTimeRangeTextOfEvents';
import ProcessActivityTooltipFields from '../EventTooltipFields/ProcessActivityTooltipFields';
import TerminationTooltipFields from '../EventTooltipFields/TerminationTooltipFields';
import DefaultTooltipFields from '../EventTooltipFields/DefaultTooltipFields';
import PolicyViolationEvent from '../EventMarker/PolicyViolationEvent';
import ProcessActivityEvent from '../EventMarker/ProcessActivityEvent';
import RestartEvent from '../EventMarker/RestartEvent';
import TerminationEvent from '../EventMarker/TerminationEvent';

type ClusteredEventsTooltipProps = {
    events: Event[];
    children: ReactElement;
};

const ClusteredEventsTooltip = ({
    events = [],
    children,
}: ClusteredEventsTooltipProps): ReactElement => {
    const timeRangeTextOfEvents = getTimeRangeTextOfEvents(events);
    const tooltipTitle = `${events.length} ${pluralize(
        'Event',
        events.length
    )} within ${timeRangeTextOfEvents}`;
    const sections = events.map(
        ({ id, type, name, args, uid, parentName, parentUid, timestamp, reason, inBaseline }) => {
            let section: ReactElement;
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
                            <ProcessActivityEvent size={15} inBaseline={inBaseline} />
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
        <Popover
            aria-label="Open to see individual processes"
            headerContent={
                <Text className="pf-u-font-weight-bold" component={TextVariants.h3}>
                    Events in this group
                </Text>
            }
            bodyContent={<DetailedTooltipContent title={tooltipTitle} body={tooltipBody} />}
        >
            {children}
        </Popover>
    );
};

export default ClusteredEventsTooltip;
