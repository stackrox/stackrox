import React from 'react';
import PropTypes from 'prop-types';

import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import Tooltip from 'Components/Tooltip';
import DetailedTooltipOverlay from 'Components/DetailedTooltipOverlay';

const EventTooltip = ({ type, name, uid, reason, timestamp, children }) => {
    const tooltipBody = (
        <>
            <div>
                <span className="font-700">Type: </span>
                <span>{eventLabels[type]}</span>
            </div>
            {uid !== null && (
                <div>
                    <span className="font-700">UID: </span>
                    <span>{uid}</span>
                </div>
            )}
            {reason !== null && (
                <div>
                    <span className="font-700">Reason: </span>
                    <span>{reason}</span>
                </div>
            )}
            <div>
                <span className="font-700">Event time: </span>
                <span>{getDateTime(timestamp)}</span>
            </div>
        </>
    );
    return (
        <Tooltip content={<DetailedTooltipOverlay title={name} body={tooltipBody} />}>
            {children}
        </Tooltip>
    );
};

EventTooltip.propTypes = {
    type: PropTypes.string.isRequired,
    name: PropTypes.string.isRequired,
    uid: PropTypes.string,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
    children: PropTypes.oneOfType([PropTypes.arrayOf(PropTypes.node), PropTypes.node]).isRequired
};

EventTooltip.defaultProps = {
    uid: null,
    reason: null
};

export default EventTooltip;
