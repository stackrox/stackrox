import React from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const TerminationTooltipFields = ({ name, reason, timestamp }) => {
    const eventTimeValue = getDateTime(timestamp);

    return (
        <>
            <TooltipFieldValue field="Name" value={name} />
            <TooltipFieldValue field="Type" value={eventLabels[eventTypes.TERMINATION]} />
            <TooltipFieldValue field="Reason" value={reason} />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

TerminationTooltipFields.propTypes = {
    name: PropTypes.string,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
};

TerminationTooltipFields.defaultProps = {
    name: null,
    reason: null,
};

export default TerminationTooltipFields;
