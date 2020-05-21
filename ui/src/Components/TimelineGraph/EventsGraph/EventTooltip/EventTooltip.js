import React from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import Tooltip from 'Components/Tooltip';
import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';
import ProcessActivityTooltipFields from './ProcessActivityTooltipFields';
import TerminationTooltipFields from './TerminationTooltipFields';
import DefaultTooltipFields from './DefaultTooltipFields';

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
    type: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    args: PropTypes.string,
    uid: PropTypes.number,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]).isRequired,
};

EventTooltip.defaultProps = {
    uid: null,
    parentName: null,
    parentUid: null,
    args: null,
    reason: null,
};

export default EventTooltip;
