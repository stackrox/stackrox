import type { ReactElement, RefObject } from 'react';
import { Tooltip } from '@patternfly/react-core';

import { eventTypes } from 'constants/timelineTypes';
import DetailedTooltipContent from 'Components/DetailedTooltipContent';
import type { Event } from './eventTypes';
import ProcessActivityTooltipFields from './EventTooltipFields/ProcessActivityTooltipFields';
import TerminationTooltipFields from './EventTooltipFields/TerminationTooltipFields';
import DefaultTooltipFields from './EventTooltipFields/DefaultTooltipFields';

export type EventTooltipProps = Event & {
    children: ReactElement;
    popoverRef: RefObject<never>;
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
    popoverRef,
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
        <Tooltip
            isContentLeftAligned
            content={<DetailedTooltipContent title={name} body={tooltipBody} />}
            triggerRef={popoverRef}
        >
            {children}
        </Tooltip>
    );
};

export default EventTooltip;
