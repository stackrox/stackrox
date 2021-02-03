import React, { ReactElement } from 'react';

import { eventTypes } from 'constants/timelineTypes';
import { Tooltip, DetailedTooltipOverlay } from '@stackrox/ui-components';
import { Event } from './eventTypes';
import ProcessActivityTooltipFields from './EventTooltipFields/ProcessActivityTooltipFields';
import TerminationTooltipFields from './EventTooltipFields/TerminationTooltipFields';
import DefaultTooltipFields from './EventTooltipFields/DefaultTooltipFields';

export type EventTooltipProps = Event & {
    children: ReactElement;
};

const EventTooltip = ({
    type,
    name,
    args,
    uid,
    parentName,
    parentUid,
    reason,
    timestamp,
    children,
}: EventTooltipProps): ReactElement => {
    let tooltipBody: ReactElement;

    switch (type) {
        case eventTypes.PROCESS_ACTIVITY:
            tooltipBody = (
                <ProcessActivityTooltipFields
                    args={args}
                    uid={uid}
                    parentName={parentName}
                    parentUid={parentUid}
                    timestamp={timestamp}
                />
            );
            break;
        case eventTypes.TERMINATION:
            tooltipBody = <TerminationTooltipFields reason={reason} timestamp={timestamp} />;
            break;
        default:
            tooltipBody = <DefaultTooltipFields type={type} timestamp={timestamp} />;
    }

    return (
        <Tooltip content={<DetailedTooltipOverlay title={name} body={tooltipBody} />}>
            {children}
        </Tooltip>
    );
};

export default EventTooltip;
