import React from 'react';
import PropTypes from 'prop-types';

import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const TerminationTooltipFields = ({ name, type, reason, timestamp }) => {
    const typeValue = eventLabels[type];
    const eventTimeValue = getDateTime(timestamp);

    return (
        <>
            <TooltipFieldValue field="Name" value={name} />
            <TooltipFieldValue field="Type" value={typeValue} />
            <TooltipFieldValue field="Reason" value={reason} />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

TerminationTooltipFields.propTypes = {
    name: PropTypes.string,
    type: PropTypes.string.isRequired,
    reason: PropTypes.string,
    timestamp: PropTypes.string.isRequired,
};

TerminationTooltipFields.defaultProps = {
    name: null,
    reason: null,
};

export default TerminationTooltipFields;
