import React from 'react';
import PropTypes from 'prop-types';

import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const DefaultTooltipFields = ({ type, timestamp }) => {
    const typeValue = eventLabels[type];
    const eventTimeValue = getDateTime(timestamp);

    return (
        <>
            <TooltipFieldValue field="Type" value={typeValue} />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

DefaultTooltipFields.propTypes = {
    type: PropTypes.string.isRequired,
    timestamp: PropTypes.string.isRequired,
};

export default DefaultTooltipFields;
