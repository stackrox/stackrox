import React from 'react';
import PropTypes from 'prop-types';

import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const TerminationTooltipFields = ({ type, reason, timestamp }) => {
    const typeValue = eventLabels[type];
    const eventTimeValue = getDateTime(timestamp);

    return (
        <>
            <TooltipFieldValue field="Type" value={typeValue} />
            <TooltipFieldValue field="Reason" value={reason} />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

TerminationTooltipFields.propTypes = {
    type: PropTypes.string.isRequired,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
};

TerminationTooltipFields.defaultProps = {
    reason: null,
};

export default TerminationTooltipFields;
