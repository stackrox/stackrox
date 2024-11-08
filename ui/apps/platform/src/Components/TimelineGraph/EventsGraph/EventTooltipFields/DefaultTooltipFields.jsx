import React from 'react';
import PropTypes from 'prop-types';

import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const DefaultTooltipFields = ({ name, type, timestamp }) => {
    const typeValue = eventLabels[type];
    const eventTimeValue = getDateTime(timestamp);

    return (
        <>
            <TooltipFieldValue field="Name" value={name} />
            <TooltipFieldValue field="Type" value={typeValue} />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

DefaultTooltipFields.propTypes = {
    name: PropTypes.string,
    type: PropTypes.string.isRequired,
    timestamp: PropTypes.string.isRequired,
};

DefaultTooltipFields.defaultProps = {
    name: null,
};

export default DefaultTooltipFields;
