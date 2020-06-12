import React from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import { eventPropTypes, defaultEventPropTypes } from 'constants/propTypes/timelinePropTypes';
import Tooltip from 'Components/Tooltip';
import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import ProcessActivityTooltipFields from './EventTooltipFields/ProcessActivityTooltipFields';
import TerminationTooltipFields from './EventTooltipFields/TerminationTooltipFields';
import DefaultTooltipFields from './EventTooltipFields/DefaultTooltipFields';

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
}) => {
    let tooltipBody = null;

    switch (type) {
        case eventTypes.PROCESS_ACTIVITY:
            tooltipBody = (
                <ProcessActivityTooltipFields
                    type={type}
                    args={args}
                    uid={uid}
                    parentName={parentName}
                    parentUid={parentUid}
                    timestamp={timestamp}
                />
            );
            break;
        case eventTypes.TERMINATION:
            tooltipBody = (
                <TerminationTooltipFields type={type} reason={reason} timestamp={timestamp} />
            );
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

EventTooltip.propTypes = {
    ...eventPropTypes,
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]).isRequired,
};

EventTooltip.defaultProps = {
    ...defaultEventPropTypes,
};

export default EventTooltip;
