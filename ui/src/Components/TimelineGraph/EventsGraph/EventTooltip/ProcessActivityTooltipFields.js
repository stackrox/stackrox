import React from 'react';
import PropTypes from 'prop-types';

import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const ProcessActivityTooltipFields = ({ type, args, uid, parentName, parentUid, timestamp }) => {
    const hasParent = parentName !== null || parentUid !== -1;
    const isParentUidUnknown = parentName !== null && parentUid === -1;
    const hasUidChanged = parentUid !== uid;

    const typeValue = eventLabels[type];
    const argsValue = args.length === 0 ? 'None' : args;
    const parentNameValue = hasParent ? parentName : 'No Parent';
    const eventTimeValue = getDateTime(timestamp);

    let uidType;
    if (hasParent && isParentUidUnknown && hasUidChanged) uidType = 'caution';
    if (hasParent && hasUidChanged) uidType = 'alert';
    else uidType = null;

    let parentUidValue;
    if (hasParent) parentUidValue = isParentUidUnknown ? 'Unknown' : parentUid;
    else parentUidValue = null;

    return (
        <>
            <TooltipFieldValue field="Type" value={typeValue} />
            <TooltipFieldValue field="Arguments" value={argsValue} />
            <TooltipFieldValue field="Parent Name" value={parentNameValue} />
            <TooltipFieldValue field="Parent UID" value={parentUidValue} />
            <TooltipFieldValue
                field="UID"
                value={uid}
                type={uidType}
                dataTestId="tooltip-uid-field-value"
            />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

ProcessActivityTooltipFields.propTypes = {
    type: PropTypes.string.isRequired,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    args: PropTypes.string,
    uid: PropTypes.number,
    timestamp: PropTypes.string.isRequired,
};

ProcessActivityTooltipFields.defaultProps = {
    uid: null,
    parentName: null,
    parentUid: null,
    args: null,
};

export default ProcessActivityTooltipFields;
