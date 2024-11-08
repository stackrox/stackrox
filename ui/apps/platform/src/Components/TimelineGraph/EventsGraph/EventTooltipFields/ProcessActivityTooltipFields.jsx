import React from 'react';
import PropTypes from 'prop-types';

import { eventTypes } from 'constants/timelineTypes';
import { eventLabels } from 'messages/timeline';
import { getDateTime } from 'utils/dateUtils';
import TooltipFieldValue from 'Components/TooltipFieldValue';

const ProcessActivityTooltipFields = ({ name, args, uid, parentName, parentUid, timestamp }) => {
    const hasParent = parentName !== null || parentUid !== -1;
    const isParentUidUnknown = parentName !== null && parentUid === -1;
    const hasUidChanged = parentUid !== uid;

    const argsValue = args === null || args.length === 0 ? 'None' : args;
    const parentNameValue = hasParent ? parentName : 'No Parent';
    const eventTimeValue = getDateTime(timestamp);

    let uidType;
    if (hasParent && isParentUidUnknown && hasUidChanged) {
        uidType = 'warning';
    }
    if (hasParent && hasUidChanged) {
        uidType = 'danger';
    } else {
        uidType = null;
    }

    let parentUidValue;
    if (hasParent) {
        parentUidValue = isParentUidUnknown ? 'Unknown' : parentUid;
    } else {
        parentUidValue = null;
    }

    return (
        <>
            <TooltipFieldValue field="Name" value={name} />
            <TooltipFieldValue field="Type" value={eventLabels[eventTypes.PROCESS_ACTIVITY]} />
            <TooltipFieldValue field="Arguments" value={argsValue} />
            <TooltipFieldValue field="Parent Name" value={parentNameValue} />
            <TooltipFieldValue field="Parent UID" value={parentUidValue} />
            <TooltipFieldValue field="UID" value={uid} type={uidType} />
            <TooltipFieldValue field="Event time" value={eventTimeValue} />
        </>
    );
};

ProcessActivityTooltipFields.propTypes = {
    name: PropTypes.string,
    parentName: PropTypes.string,
    parentUid: PropTypes.number,
    args: PropTypes.string,
    uid: PropTypes.number,
    timestamp: PropTypes.string.isRequired,
};

ProcessActivityTooltipFields.defaultProps = {
    name: null,
    uid: null,
    parentName: null,
    parentUid: null,
    args: null,
};

export default ProcessActivityTooltipFields;
